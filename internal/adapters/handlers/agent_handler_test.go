package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MiltonJ23/Kliops/internal/core/services"
)

type stubOrchestrator struct {
	response string
	err      error
}

func (s *stubOrchestrator) Run(_ context.Context, _ string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.response, nil
}

func newValidAgentRequestBody() []byte {
	body, _ := json.Marshal(map[string]string{
		"dce_content":  strings.Repeat("Exigence technique critique. ", 3),
		"project_name": "Projet Elite",
		"target_email": "team@example.com",
		"client_name":  "Client Premium",
	})
	return body
}

func TestAgentHandlerHandleQuerySuccess(t *testing.T) {
	h := NewAgentHandler(services.NewAgentService(&stubOrchestrator{response: "https://docs.example.com/doc/123"}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/ask", bytes.NewReader(newValidAgentRequestBody()))
	rec := httptest.NewRecorder()

	h.HandleQuery(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", rec.Code, rec.Body.String())
	}
}

func TestAgentHandlerHandleQueryInvalidJSON(t *testing.T) {
	h := NewAgentHandler(services.NewAgentService(&stubOrchestrator{response: "unused"}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/ask", bytes.NewBufferString("{"))
	rec := httptest.NewRecorder()

	h.HandleQuery(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAgentHandlerHandleQueryValidationError(t *testing.T) {
	h := NewAgentHandler(services.NewAgentService(&stubOrchestrator{response: "unused"}))
	badBody := []byte(`{"dce_content":"short","project_name":"P","target_email":"bad","client_name":"Client"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/ask", bytes.NewReader(badBody))
	rec := httptest.NewRecorder()

	h.HandleQuery(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d with body %s", rec.Code, rec.Body.String())
	}
}

func TestAgentHandlerHandleQueryTimeout(t *testing.T) {
	h := NewAgentHandler(services.NewAgentService(&stubOrchestrator{err: context.DeadlineExceeded}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/ask", bytes.NewReader(newValidAgentRequestBody()))
	rec := httptest.NewRecorder()

	h.HandleQuery(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected 504, got %d with body %s", rec.Code, rec.Body.String())
	}
}
