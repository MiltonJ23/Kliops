package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/MiltonJ23/Kliops/internal/core/services"
)

// agentTimeout is the hard deadline for a single agent run.
// A full pipeline (RAG + pricing + doc generation) can legitimately take several
// minutes with a local Gemma instance. Adjust based on your Ollama hardware.
const agentTimeout = 10 * time.Minute

// maxRequestBodyBytes limits the JSON body to prevent memory exhaustion.
// 10 MB covers any realistic DCE text payload.
const maxRequestBodyBytes = 10 << 20

// ─── Handler ─────────────────────────────────────────────────────────────────

// AgentHandler handles HTTP requests for the agentic pipeline.
type AgentHandler struct {
	svc     *services.AgentService
	timeout time.Duration
}

// NewAgentHandler constructs a production-ready agent handler.
func NewAgentHandler(svc *services.AgentService) *AgentHandler {
	if svc == nil {
		panic("agenthandler: AgentService must not be nil")
	}
	return &AgentHandler{
		svc:     svc,
		timeout: agentTimeout,
	}
}

// ─── Request / Response DTOs ─────────────────────────────────────────────────

// askRequest is the JSON body accepted by POST /api/v1/agent/ask.
type askRequest struct {
	// DCEContent is the plain-text content of the DCE document.
	// The client is responsible for PDF-to-text conversion before posting.
	DCEContent  string `json:"dce_content"`
	ProjectName string `json:"project_name"`
	TargetEmail string `json:"target_email"`
	ClientName  string `json:"client_name"`
}

// askResponse is the JSON body returned on a successful run.
type askResponse struct {
	Message    string `json:"message"`
	DurationMs int64  `json:"duration_ms"`
}

// apiError is the JSON body returned on any error path.
type apiError struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}

// ─── Handler method ──────────────────────────────────────────────────────────

// HandleQuery processes a tender request through the agentic pipeline.
//
// POST /api/v1/agent/ask
//
//	Content-Type: application/json
//	X-API-KEY: <secret>
//
// Response codes:
//
//	200 — document generated, body contains {message, duration_ms}
//	400 — invalid or malformed request
//	408 — request timeout (client closed connection)
//	413 — request body too large
//	422 — request is well-formed but failed domain validation
//	504 — agent did not finish within the server timeout
//	500 — unexpected internal error
func (h *AgentHandler) HandleQuery(w http.ResponseWriter, r *http.Request) {
	reqID := requestID(r)
	logger := prefixLogger(reqID)

	// ── Parse body ────────────────────────────────────────────────────────────
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

	var req askRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			logger("body exceeds %d bytes", maxRequestBodyBytes)
			writeError(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		logger("JSON decode error: %v", err)
		writeError(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// ── Build service request ─────────────────────────────────────────────────
	svcReq := &services.TenderRequest{
		DCEContent:  strings.TrimSpace(req.DCEContent),
		ProjectName: strings.TrimSpace(req.ProjectName),
		TargetEmail: strings.TrimSpace(req.TargetEmail),
		ClientName:  strings.TrimSpace(req.ClientName),
	}

	logger("received tender request — project=%q email=%q dce_len=%d",
		svcReq.ProjectName, svcReq.TargetEmail, len(svcReq.DCEContent))

	// ── Apply server-side timeout ─────────────────────────────────────────────
	// We layer our own deadline on top of the request context so that:
	// (a) the agent loop is bounded regardless of client keep-alive settings, and
	// (b) the client disconnect (r.Context cancellation) still propagates.
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	// ── Run pipeline ──────────────────────────────────────────────────────────
	start := time.Now()
	result, err := h.svc.ProcessTender(ctx, svcReq)
	if err != nil {
		elapsed := time.Since(start).Milliseconds()
		h.handleServiceError(w, err, elapsed, logger)
		return
	}

	logger("pipeline completed — %dms", result.DurationMs)

	writeJSON(w, http.StatusOK, askResponse{
		Message:    result.Message,
		DurationMs: result.DurationMs,
	})
}

// ─── Error classification ─────────────────────────────────────────────────────

func (h *AgentHandler) handleServiceError(w http.ResponseWriter, err error, elapsedMs int64, logger func(string, ...any)) {
	switch {

	case errors.Is(err, context.DeadlineExceeded):
		logger("agent timeout after %dms", elapsedMs)
		writeError(w,
			fmt.Sprintf("le pipeline agent a dépassé le délai de %v — réessayez avec un DCE plus court ou contactez l'administrateur", h.timeout),
			http.StatusGatewayTimeout,
		)

	case errors.Is(err, context.Canceled):
		// Client disconnected — nothing to send back.
		logger("request cancelled by client after %dms", elapsedMs)

	case errors.Is(err, services.ErrInvalidRequest):
		// Safe to surface — these are domain-level validation messages.
		logger("invalid request: %v", err)
		writeError(w, strings.TrimPrefix(err.Error(), "requête invalide: "), http.StatusUnprocessableEntity)

	case errors.Is(err, services.ErrPipelineFailed):
		// Orchestration error — log full detail, return generic message.
		logger("pipeline error after %dms: %v", elapsedMs, err)
		writeError(w, "le pipeline de génération a échoué — veuillez réessayer", http.StatusInternalServerError)

	default:
		logger("unexpected error after %dms: %v", elapsedMs, err)
		writeError(w, "erreur interne inattendue", http.StatusInternalServerError)
	}
}

// ─── Response helpers ─────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("[AgentHandler] failed to encode response: %v", err)
	}
}

func writeError(w http.ResponseWriter, message string, status int) {
	writeJSON(w, status, apiError{Error: message, Code: status})
}

// ─── Logging helpers ─────────────────────────────────────────────────────────

// requestID extracts a correlation ID from the request, falling back to a
// timestamp-based surrogate. Works with any gateway that injects X-Request-ID.
func requestID(r *http.Request) string {
	if id := r.Header.Get("X-Request-ID"); id != "" {
		return id
	}
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

func prefixLogger(reqID string) func(string, ...any) {
	return func(format string, args ...any) {
		log.Printf("[AgentHandler][%s] %s", reqID, fmt.Sprintf(format, args...))
	}
}
