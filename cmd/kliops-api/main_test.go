package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MiltonJ23/Kliops/internal/adapters/handlers"
	"github.com/MiltonJ23/Kliops/internal/core/services"
)

type fakeStorage struct{}

func (fakeStorage) Upload(context.Context, string, string, io.Reader, int64, string) (string, error) {
	return "minio://bucket/object", nil
}

func (fakeStorage) Delete(context.Context, string, string) error {
	return nil
}

func (fakeStorage) DownloadStream(context.Context, string, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("template")), nil
}

type fakeMainOrchestrator struct{}

func (fakeMainOrchestrator) Run(context.Context, string) (string, error) {
	return "https://docs.example.com/generated", nil
}

func TestLoadConfigUsesExpectedDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/kliops")
	t.Setenv("MINIO_ENDPOINT", "localhost:9000")
	t.Setenv("MINIO_ROOT_USER", "minio")
	t.Setenv("MINIO_ROOT_PASSWORD", "password")
	t.Setenv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	t.Setenv("QDRANT_ADDR", "localhost:6334")
	t.Setenv("API_KEY_SECRET", "secret")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if cfg.OllamaChatModel != "gemma4:e4b" {
		t.Fatalf("expected default Ollama chat model, got %q", cfg.OllamaChatModel)
	}
	if cfg.OllamaEmbeddingModel != "mxbai-embed-large" {
		t.Fatalf("expected default embedding model, got %q", cfg.OllamaEmbeddingModel)
	}
}

func TestRoutesProtectAndStripAPIV1Prefix(t *testing.T) {
	t.Setenv("API_KEY_SECRET", "secret")

	app := &application{startedAt: time.Unix(0, 0).UTC()}
	gatewayHandler := handlers.NewGatewayHandler(fakeStorage{}, services.NewPricingService())
	ingestionHandler := handlers.NewIngestionHandler(nil, fakeStorage{})
	agentHandler := handlers.NewAgentHandler(services.NewAgentService(fakeMainOrchestrator{}))
	mux := app.routes(gatewayHandler, ingestionHandler, agentHandler)

	body := []byte(`{"dce_content":"` + strings.Repeat("Exigence technique critique. ", 3) + `","project_name":"Projet Elite","target_email":"team@example.com","client_name":"Client"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/ask", bytes.NewReader(body))
	req.Header.Set("X-API-KEY", "secret")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", rec.Code, rec.Body.String())
	}
}
