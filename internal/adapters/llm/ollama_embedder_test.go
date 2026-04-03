package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Mock HTTP transport (avoids real TCP connections)
// ---------------------------------------------------------------------------

// roundTripFunc implements http.RoundTripper as a simple function value.
type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// makeJSONResponse builds an *http.Response with a JSON body and given status.
func makeJSONResponse(status int, body interface{}) *http.Response {
	b, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(b)),
	}
}

// makeRawResponse builds an *http.Response with a raw string body and given status.
func makeRawResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"text/plain"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// embedderWithTransport returns an OllamaEmbedder backed by the given transport.
func embedderWithTransport(transport http.RoundTripper, model string) *OllamaEmbedder {
	e := NewOllamaEmbedder("http://ollama-server:11434", model)
	e.Client = &http.Client{Transport: transport}
	return e
}

// ---------------------------------------------------------------------------
// Constructor tests
// ---------------------------------------------------------------------------

func TestNewOllamaEmbedderFields(t *testing.T) {
	e := NewOllamaEmbedder("http://localhost:11434", "mxbai-embed-large")
	if e.BaseUrl != "http://localhost:11434" {
		t.Errorf("unexpected BaseUrl: %q", e.BaseUrl)
	}
	if e.Model != "mxbai-embed-large" {
		t.Errorf("unexpected Model: %q", e.Model)
	}
	if e.Client == nil {
		t.Error("expected non-nil http.Client")
	}
}

func TestNewOllamaEmbedderEmptyStrings(t *testing.T) {
	e := NewOllamaEmbedder("", "")
	if e.BaseUrl != "" {
		t.Errorf("expected empty BaseUrl, got %q", e.BaseUrl)
	}
	if e.Model != "" {
		t.Errorf("expected empty Model, got %q", e.Model)
	}
}

// ---------------------------------------------------------------------------
// CreateEmbedding – happy path
// ---------------------------------------------------------------------------

func TestCreateEmbeddingSuccess(t *testing.T) {
	wantEmbedding := []float64{0.1, 0.2, 0.3, 0.4}
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return makeJSONResponse(http.StatusOK, EmbeddingResponse{Embedding: wantEmbedding}), nil
	})
	e := embedderWithTransport(transport, "test-model")

	got, err := e.CreateEmbedding(context.Background(), "some technical requirement")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(wantEmbedding) {
		t.Fatalf("expected vector length %d, got %d", len(wantEmbedding), len(got))
	}
	for i, v := range got {
		want := float32(wantEmbedding[i])
		if v != want {
			t.Errorf("vector[%d]: expected %f, got %f", i, want, v)
		}
	}
}

func TestCreateEmbeddingConvertsFloat64ToFloat32(t *testing.T) {
	wantEmbedding := []float64{1.0, 0.5, 0.25, 0.125}
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return makeJSONResponse(http.StatusOK, EmbeddingResponse{Embedding: wantEmbedding}), nil
	})
	e := embedderWithTransport(transport, "test-model")

	got, err := e.CreateEmbedding(context.Background(), "text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, v := range got {
		expected := float32(wantEmbedding[i])
		if v != expected {
			t.Errorf("vector[%d]: float64->float32: expected %v, got %v", i, expected, v)
		}
	}
}

func TestCreateEmbeddingEmptyVector(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return makeJSONResponse(http.StatusOK, EmbeddingResponse{Embedding: []float64{}}), nil
	})
	e := embedderWithTransport(transport, "test-model")

	got, err := e.CreateEmbedding(context.Background(), "empty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty vector, got length %d", len(got))
	}
}

func TestCreateEmbeddingLargeVector(t *testing.T) {
	large := make([]float64, 1024)
	for i := range large {
		large[i] = float64(i) / 1024.0
	}
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return makeJSONResponse(http.StatusOK, EmbeddingResponse{Embedding: large}), nil
	})
	e := embedderWithTransport(transport, "large-model")

	got, err := e.CreateEmbedding(context.Background(), "large vector test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1024 {
		t.Errorf("expected 1024-dim vector, got %d", len(got))
	}
}

func TestCreateEmbeddingSendsModelAndPrompt(t *testing.T) {
	var captured EmbeddingRequest
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		_ = json.NewDecoder(req.Body).Decode(&captured)
		return makeJSONResponse(http.StatusOK, EmbeddingResponse{Embedding: []float64{0.5}}), nil
	})
	e := embedderWithTransport(transport, "mxbai-embed-large")

	_, err := e.CreateEmbedding(context.Background(), "ma requête technique")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.Model != "mxbai-embed-large" {
		t.Errorf("expected model mxbai-embed-large in request, got %q", captured.Model)
	}
	if captured.Prompt != "ma requête technique" {
		t.Errorf("expected prompt %q, got %q", "ma requête technique", captured.Prompt)
	}
}

func TestCreateEmbeddingURLPath(t *testing.T) {
	var receivedURL string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		receivedURL = req.URL.Path
		return makeJSONResponse(http.StatusOK, EmbeddingResponse{Embedding: []float64{0.1}}), nil
	})
	e := embedderWithTransport(transport, "model")

	_, err := e.CreateEmbedding(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedURL != "/api/embeddings" {
		t.Errorf("expected path /api/embeddings, got %q", receivedURL)
	}
}

func TestCreateEmbeddingHTTPMethod(t *testing.T) {
	var receivedMethod string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		receivedMethod = req.Method
		return makeJSONResponse(http.StatusOK, EmbeddingResponse{Embedding: []float64{0.1}}), nil
	})
	e := embedderWithTransport(transport, "model")

	_, _ = e.CreateEmbedding(context.Background(), "test")
	if receivedMethod != http.MethodPost {
		t.Errorf("expected POST method, got %q", receivedMethod)
	}
}

func TestCreateEmbeddingContentTypeHeader(t *testing.T) {
	var receivedCT string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		receivedCT = req.Header.Get("content-type")
		return makeJSONResponse(http.StatusOK, EmbeddingResponse{Embedding: []float64{0.1}}), nil
	})
	e := embedderWithTransport(transport, "model")

	_, _ = e.CreateEmbedding(context.Background(), "test")
	if receivedCT != "application/json" {
		t.Errorf("expected content-type application/json, got %q", receivedCT)
	}
}

// ---------------------------------------------------------------------------
// CreateEmbedding – error paths
// ---------------------------------------------------------------------------

func TestCreateEmbeddingNon200Status(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"500 internal server error", http.StatusInternalServerError},
		{"404 not found", http.StatusNotFound},
		{"401 unauthorized", http.StatusUnauthorized},
		{"503 service unavailable", http.StatusServiceUnavailable},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return makeRawResponse(tc.status, "server error"), nil
			})
			e := embedderWithTransport(transport, "model")

			_, err := e.CreateEmbedding(context.Background(), "text")
			if err == nil {
				t.Fatalf("expected error for status %d, got nil", tc.status)
			}
			if !strings.Contains(err.Error(), fmt.Sprintf("%d", tc.status)) {
				t.Errorf("expected error to mention status %d, got: %q", tc.status, err.Error())
			}
		})
	}
}

func TestCreateEmbeddingConnectionRefused(t *testing.T) {
	// Point to a port that isn't listening (no transport override — uses real TCP)
	e := NewOllamaEmbedder("http://127.0.0.1:19999", "model")
	_, err := e.CreateEmbedding(context.Background(), "text")
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to connect to Ollama") {
		t.Errorf("expected connection error message, got: %q", err.Error())
	}
}

func TestCreateEmbeddingTransportError(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("transport failure")
	})
	e := embedderWithTransport(transport, "model")

	_, err := e.CreateEmbedding(context.Background(), "text")
	if err == nil {
		t.Fatal("expected transport error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to connect to Ollama") {
		t.Errorf("expected connection error wrapper, got: %q", err.Error())
	}
}

func TestCreateEmbeddingInvalidJSONResponse(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader("not-valid-json")),
		}, nil
	})
	e := embedderWithTransport(transport, "model")

	_, err := e.CreateEmbedding(context.Background(), "text")
	if err == nil {
		t.Fatal("expected JSON decode error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to decode") {
		t.Errorf("expected decode error message, got: %q", err.Error())
	}
}

func TestCreateEmbeddingContextCancelled(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		// Simulate context cancellation being honoured
		<-req.Context().Done()
		return nil, req.Context().Err()
	})
	e := embedderWithTransport(transport, "model")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := e.CreateEmbedding(ctx, "text")
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}

func TestCreateEmbeddingInvalidBaseURL(t *testing.T) {
	// An invalid URL scheme causes request creation to fail before transport is used
	e := NewOllamaEmbedder("://invalid-url", "model")
	_, err := e.CreateEmbedding(context.Background(), "text")
	if err == nil {
		t.Fatal("expected error for invalid base URL, got nil")
	}
}

func TestCreateEmbeddingErrorBodyIncludedInMessage(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return makeRawResponse(http.StatusBadRequest, "model not found"), nil
	})
	e := embedderWithTransport(transport, "unknown-model")

	_, err := e.CreateEmbedding(context.Background(), "text")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "model not found") {
		t.Errorf("expected error to contain response body, got: %q", err.Error())
	}
}

func TestCreateEmbeddingBaseURLUsed(t *testing.T) {
	var receivedHost string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		receivedHost = req.URL.Host
		return makeJSONResponse(http.StatusOK, EmbeddingResponse{Embedding: []float64{0.1}}), nil
	})
	e := NewOllamaEmbedder("http://my-ollama-host:11434", "model")
	e.Client = &http.Client{Transport: transport}

	_, err := e.CreateEmbedding(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedHost != "my-ollama-host:11434" {
		t.Errorf("expected host my-ollama-host:11434, got %q", receivedHost)
	}
}