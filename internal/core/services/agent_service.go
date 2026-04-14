package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

// ─── Port interface ───────────────────────────────────────────────────────────

// AgentOrchestrator is the port the application service uses to run the
// agentic pipeline. The implementation lives in adapters/agent to prevent
// circular imports between core and adapters.
type AgentOrchestrator interface {
	Run(ctx context.Context, prompt string) (string, error)
}

// ─── Request / Response types ─────────────────────────────────────────────────

// TenderRequest carries every piece of information needed to produce a mémoire
// technique. All fields are validated before reaching the orchestrator.
type TenderRequest struct {
	// DCEContent is the raw extracted text of the Dossier de Consultation.
	// The caller is responsible for parsing the PDF and passing the plain text.
	DCEContent string `json:"dce_content"`

	// ProjectName becomes the document title and is injected into the template.
	ProjectName string `json:"project_name"`

	// TargetEmail is the address to which the generated DOCX will be shared.
	TargetEmail string `json:"target_email"`

	// ClientName is the maître d'ouvrage, injected into the template header.
	ClientName string `json:"client_name"`
}

// TenderResponse wraps the output of a successful agent run.
type TenderResponse struct {
	// Message is the raw text returned by the supervisor agent — normally the
	// document URL or a human-readable confirmation.
	Message string `json:"message"`

	// DurationMs is the wall-clock time the full agentic pipeline took.
	DurationMs int64 `json:"duration_ms"`
}

// ─── Sentinel errors ─────────────────────────────────────────────────────────

var (
	ErrInvalidRequest = errors.New("requête invalide")
	ErrPipelineFailed = errors.New("pipeline agent échoué")
)

// ─── Service ─────────────────────────────────────────────────────────────────

// AgentService is the application-level orchestration service.
// It owns validation, prompt construction, logging, and delegates the actual
// LLM loop to the AgentOrchestrator (injected as an interface to decouple the
// core from the adapter layer).
type AgentService struct {
	orchestrator AgentOrchestrator
}

// NewAgentService constructs the service. Pass the concrete orchestrator from
// adapters/agent as the AgentOrchestrator interface.
func NewAgentService(orch AgentOrchestrator) *AgentService {
	if orch == nil {
		panic("agentservice: orchestrator must not be nil")
	}
	return &AgentService{orchestrator: orch}
}

// ProcessTender validates the request, builds the LLM prompt, runs the
// agentic pipeline, and returns a structured response.
func (s *AgentService) ProcessTender(ctx context.Context, req *TenderRequest) (*TenderResponse, error) {
	if err := validateTenderRequest(req); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}

	prompt := buildPrompt(req)

	start := time.Now()
	log.Printf("[AgentService] Démarrage — projet=%q email=%q dce_len=%d",
		req.ProjectName, req.TargetEmail, len(req.DCEContent))

	raw, err := s.orchestrator.Run(ctx, prompt)
	if err != nil {
		elapsed := time.Since(start).Milliseconds()
		log.Printf("[AgentService] Échec après %dms — %v", elapsed, err)
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", ErrPipelineFailed, err)
	}

	elapsed := time.Since(start).Milliseconds()
	log.Printf("[AgentService] Terminé en %dms", elapsed)

	return &TenderResponse{
		Message:    raw,
		DurationMs: elapsed,
	}, nil
}

// ─── Validation ───────────────────────────────────────────────────────────────

const (
	maxDCELength = 300_000 // ~300 kB of plain text — well above a typical DCE
	minDCELength = 50      // Reject obviously empty submissions
)

func validateTenderRequest(req *TenderRequest) error {
	if req == nil {
		return errors.New("request is nil")
	}

	req.DCEContent = strings.TrimSpace(req.DCEContent)
	req.ProjectName = strings.TrimSpace(req.ProjectName)
	req.TargetEmail = strings.TrimSpace(req.TargetEmail)
	req.ClientName = strings.TrimSpace(req.ClientName)

	if len(req.DCEContent) < minDCELength {
		return fmt.Errorf("dce_content trop court (%d caractères, minimum %d)", len(req.DCEContent), minDCELength)
	}
	if len(req.DCEContent) > maxDCELength {
		return fmt.Errorf("dce_content trop long (%d caractères, maximum %d)", len(req.DCEContent), maxDCELength)
	}
	if req.ProjectName == "" {
		return errors.New("project_name est obligatoire")
	}
	if len(req.ProjectName) > 200 {
		return errors.New("project_name dépasse 200 caractères")
	}
	if req.TargetEmail == "" {
		return errors.New("target_email est obligatoire")
	}
	if !isValidEmail(req.TargetEmail) {
		return fmt.Errorf("target_email invalide: %q", req.TargetEmail)
	}

	return nil
}

// isValidEmail is a lightweight structural check — no regex dependency.
func isValidEmail(email string) bool {
	at := strings.LastIndex(email, "@")
	if at < 1 || at == len(email)-1 {
		return false
	}
	domain := email[at+1:]
	dot := strings.LastIndex(domain, ".")
	return dot > 0 && dot < len(domain)-1
}

// ─── Prompt builder ───────────────────────────────────────────────────────────

// buildPrompt constructs the user-turn message sent to the supervisor agent.
// Keeping prompt construction here (rather than in the orchestrator) means we
// can unit-test it independently and iterate quickly without touching the LLM layer.
func buildPrompt(req *TenderRequest) string {
	var b strings.Builder

	b.WriteString("Traite l'appel d'offres ci-dessous et génère le mémoire technique complet.\n\n")

	b.WriteString("━━━ CONTEXTE DU PROJET ━━━\n")
	fmt.Fprintf(&b, "Projet       : %s\n", req.ProjectName)
	fmt.Fprintf(&b, "Client       : %s\n", req.ClientName)
	fmt.Fprintf(&b, "Destinataire : %s\n\n", req.TargetEmail)

	b.WriteString("━━━ CONTRAINTES DE GÉNÉRATION ━━━\n")
	fmt.Fprintf(&b, "• Utilise project_name = %q dans generate_document.\n", req.ProjectName)
	fmt.Fprintf(&b, "• Utilise target_email = %q dans generate_document.\n", req.TargetEmail)
	b.WriteString("• Remplis TOUS les placeholders du template SITINFRA (NOM_CLIENT, OBJET_MARCHE, etc.).\n")
	b.WriteString("• Si un code article est absent de la mercuriale, indique-le avec la valeur \"PRIX INDISPONIBLE\".\n\n")

	b.WriteString("━━━ CONTENU DU DOSSIER DE CONSULTATION (DCE) ━━━\n")
	b.WriteString(req.DCEContent)
	b.WriteString("\n━━━ FIN DU DCE ━━━\n")

	return b.String()
}
