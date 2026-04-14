package services

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type stubAgentOrchestrator struct {
	response string
	err      error
	prompt   string
}

func (s *stubAgentOrchestrator) Run(_ context.Context, prompt string) (string, error) {
	s.prompt = prompt
	if s.err != nil {
		return "", s.err
	}
	return s.response, nil
}

func validTenderRequest() *TenderRequest {
	return &TenderRequest{
		DCEContent:  strings.Repeat("Exigence technique ", 4),
		ProjectName: "Projet Elite",
		TargetEmail: "team@example.com",
		ClientName:  "Client Premium",
	}
}

func TestAgentServiceProcessTenderSuccess(t *testing.T) {
	orch := &stubAgentOrchestrator{response: "https://docs.example.com/doc/123"}
	svc := NewAgentService(orch)

	resp, err := svc.ProcessTender(context.Background(), validTenderRequest())
	if err != nil {
		t.Fatalf("ProcessTender() error = %v", err)
	}
	if resp.Message != orch.response {
		t.Fatalf("unexpected response message: %q", resp.Message)
	}
	if resp.DurationMs < 0 {
		t.Fatalf("duration should never be negative: %d", resp.DurationMs)
	}
	if !strings.Contains(orch.prompt, "Projet       : Projet Elite") {
		t.Fatalf("expected built prompt to contain project name, got %q", orch.prompt)
	}
}

func TestAgentServiceProcessTenderPropagatesContextErrors(t *testing.T) {
	orch := &stubAgentOrchestrator{err: context.DeadlineExceeded}
	svc := NewAgentService(orch)

	_, err := svc.ProcessTender(context.Background(), validTenderRequest())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
}

func TestAgentServiceProcessTenderValidatesInput(t *testing.T) {
	orch := &stubAgentOrchestrator{response: "unused"}
	svc := NewAgentService(orch)

	_, err := svc.ProcessTender(context.Background(), &TenderRequest{
		DCEContent:  "too short",
		ProjectName: "Projet Elite",
		TargetEmail: "invalid-email",
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected ErrInvalidRequest, got %v", err)
	}
}
