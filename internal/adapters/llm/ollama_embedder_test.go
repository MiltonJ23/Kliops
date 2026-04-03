package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// roundTripFunc allows a function to be used as an http.RoundTripper.
// ---------------------------------------------------------------------------

type roundTripFunc func(r *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// newEmbedderWithTransport creates an OllamaEmbedder whose HTTP client uses the given transport.
func newEmbedderWithTransport(baseURL, model string, rt http.RoundTripper) *OllamaEmbedder {
	return &OllamaEmbedder{
		BaseUrl: baseURL,
		Model:   model,
		Client:  &http.Client{Transport: rt},
	}
}

// jsonResponse builds an *http.Response whose body contains the JSON-encoding of v.
func jsonResponse(statusCode int, v interface{}) *http.Response {
	body, _ := json.Marshal(v)
	return &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

// plainResponse builds an *http.Response with a plain-text body.
func plainResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// ---------------------------------------------------------------------------
// NewOllamaEmbedder
// ---------------------------------------------------------------------------

func TestNewOllamaEmbedder(t *testing.T) {
	e := NewOllamaEmbedder("http://localhost:11434", "mxbai-embed-large")
	if e.BaseUrl != "http://localhost:11434" {
		t.Errorf("expected BaseUrl %q, got %q", "http://localhost:11434", e.BaseUrl)
	}
	if e.Model != "mxbai-embed-large" {
		t.Errorf("expected Model %q, got %q", "mxbai-embed-large", e.Model)
	}
	if e.Client == nil {
		t.Error("expected non-nil http.Client")
	}
}

func TestNewOllamaEmbedderEmptyFields(t *testing.T) {
	e := NewOllamaEmbedder("", "")
	if e.BaseUrl != "" {
		t.Errorf("expected empty BaseUrl, got %q", e.BaseUrl)
	}
	if e.Model != "" {
		t.Errorf("expected empty Model, got %q", e.Model)
	}
	if e.Client == nil {
		t.Error("expected non-nil http.Client even with empty fields")
	}
}

// ---------------------------------------------------------------------------
// CreateEmbedding — success paths
// ---------------------------------------------------------------------------

func TestCreateEmbedding_Success(t *testing.T) {
	expectedEmbedding := []float64{0.1, 0.2, 0.3, 0.4, 0.5}

	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		// Verify method and path.
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/embeddings" {
			t.Errorf("expected path /api/embeddings, got %s", r.URL.Path)
		}
		if ct := r.Header.Get("content-type"); ct != "application/json" {
			t.Errorf("expected content-type application/json, got %q", ct)
		}

		// Verify request body.
		var req EmbeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if req.Model != "test-model" {
			t.Errorf("expected model %q in body, got %q", "test-model", req.Model)
		}
		if req.Prompt != "hello world" {
			t.Errorf("expected prompt %q in body, got %q", "hello world", req.Prompt)
		}

		return jsonResponse(http.StatusOK, EmbeddingResponse{Embedding: expectedEmbedding}), nil
	})

	e := newEmbedderWithTransport("http://ollama", "test-model", rt)
	vec, err := e.CreateEmbedding(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vec) != len(expectedEmbedding) {
		t.Fatalf("expected vector length %d, got %d", len(expectedEmbedding), len(vec))
	}
	for i, v := range expectedEmbedding {
		if vec[i] != float32(v) {
			t.Errorf("vector[%d]: expected %f, got %f", i, float32(v), vec[i])
		}
	}
}

func TestCreateEmbedding_Float32Conversion(t *testing.T) {
	// Verify float64 → float32 narrowing for each element.
	embedding := []float64{1.0, 0.5, 0.25, 0.125}

	rt := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, EmbeddingResponse{Embedding: embedding}), nil
	})

	e := newEmbedderWithTransport("http://ollama", "model", rt)
	vec, err := e.CreateEmbedding(context.Background(), "text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, v := range embedding {
		want := float32(v)
		if vec[i] != want {
			t.Errorf("vector[%d]: expected %f, got %f", i, want, vec[i])
		}
	}
}

func TestCreateEmbedding_EmptyEmbedding(t *testing.T) {
	rt := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, EmbeddingResponse{Embedding: []float64{}}), nil
	})

	e := newEmbedderWithTransport("http://ollama", "model", rt)
	vec, err := e.CreateEmbedding(context.Background(), "text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vec) != 0 {
		t.Errorf("expected empty vector, got length %d", len(vec))
	}
}

// ---------------------------------------------------------------------------
// CreateEmbedding — error paths
// ---------------------------------------------------------------------------

func TestCreateEmbedding_ServerReturnsNonOK(t *testing.T) {
	rt := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return plainResponse(http.StatusNotFound, "model not found"), nil
	})

	e := newEmbedderWithTransport("http://ollama", "missing-model", rt)
	_, err := e.CreateEmbedding(context.Background(), "text")
	if err == nil {
		t.Fatal("expected error for non-200 response, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error to contain HTTP status 404, got: %v", err)
	}
}

func TestCreateEmbedding_ServerReturns500(t *testing.T) {
	rt := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return plainResponse(http.StatusInternalServerError, "internal server error"), nil
	})

	e := newEmbedderWithTransport("http://ollama", "model", rt)
	_, err := e.CreateEmbedding(context.Background(), "text")
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to mention status 500, got: %v", err)
	}
}

func TestCreateEmbedding_InvalidJSONResponse(t *testing.T) {
	rt := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("not valid json {{{")),
		}, nil
	})

	e := newEmbedderWithTransport("http://ollama", "model", rt)
	_, err := e.CreateEmbedding(context.Background(), "text")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "failed to decode") {
		t.Errorf("expected decode error message, got: %v", err)
	}
}

func TestCreateEmbedding_TransportError(t *testing.T) {
	transportErr := errors.New("connection refused")
	rt := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return nil, transportErr
	})

	e := newEmbedderWithTransport("http://ollama", "model", rt)
	_, err := e.CreateEmbedding(context.Background(), "text")
	if err == nil {
		t.Fatal("expected transport error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to connect to Ollama") {
		t.Errorf("expected connection error message, got: %v", err)
	}
}

func TestCreateEmbedding_ConnectionRefused(t *testing.T) {
	// Use a real HTTP client pointed at an address that is guaranteed unreachable.
	e := NewOllamaEmbedder("http://127.0.0.1:1", "model")
	_, err := e.CreateEmbedding(context.Background(), "text")
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to connect to Ollama") {
		t.Errorf("expected connection error message, got: %v", err)
	}
}

func TestCreateEmbedding_CancelledContext(t *testing.T) {
	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		// Honour the context cancellation.
		select {
		case <-r.Context().Done():
			return nil, r.Context().Err()
		default:
			return jsonResponse(http.StatusOK, EmbeddingResponse{Embedding: []float64{1.0}}), nil
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the call

	e := newEmbedderWithTransport("http://ollama", "model", rt)
	_, err := e.CreateEmbedding(ctx, "text")
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}

// ---------------------------------------------------------------------------
// CreateEmbedding — request content validation
// ---------------------------------------------------------------------------

func TestCreateEmbedding_RequestBodyContainsModelAndPrompt(t *testing.T) {
	var captured EmbeddingRequest

	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		json.NewDecoder(r.Body).Decode(&captured)
		return jsonResponse(http.StatusOK, EmbeddingResponse{Embedding: []float64{0.42}}), nil
	})

	e := newEmbedderWithTransport("http://ollama", "nomic-embed", rt)
	_, err := e.CreateEmbedding(context.Background(), "unique prompt text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.Model != "nomic-embed" {
		t.Errorf("expected model %q in request body, got %q", "nomic-embed", captured.Model)
	}
	if captured.Prompt != "unique prompt text" {
		t.Errorf("expected prompt %q in request body, got %q", "unique prompt text", captured.Prompt)
	}
}

func TestCreateEmbedding_URLContainsBaseURLAndPath(t *testing.T) {
	var capturedURL string

	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		capturedURL = r.URL.String()
		return jsonResponse(http.StatusOK, EmbeddingResponse{Embedding: []float64{0.1}}), nil
	})

	e := newEmbedderWithTransport("http://myollama:11434", "model", rt)
	_, err := e.CreateEmbedding(context.Background(), "text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(capturedURL, "/api/embeddings") {
		t.Errorf("expected URL to end with /api/embeddings, got %q", capturedURL)
	}
}

// ---------------------------------------------------------------------------
// CreateEmbedding — boundary / regression
// ---------------------------------------------------------------------------

func TestCreateEmbedding_LargeVector(t *testing.T) {
	const dim = 1024
	embedding := make([]float64, dim)
	for i := range embedding {
		embedding[i] = float64(i) * 0.001
	}

	rt := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, EmbeddingResponse{Embedding: embedding}), nil
	})

	e := newEmbedderWithTransport("http://ollama", "large-model", rt)
	vec, err := e.CreateEmbedding(context.Background(), "large embedding test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vec) != dim {
		t.Errorf("expected vector length %d, got %d", dim, len(vec))
	}
	want := float32(float64(dim-1) * 0.001)
	if vec[dim-1] != want {
		t.Errorf("vector[%d]: expected %f, got %f", dim-1, want, vec[dim-1])
	}
}

func TestCreateEmbedding_SingleElementVector(t *testing.T) {
	rt := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, EmbeddingResponse{Embedding: []float64{0.99}}), nil
	})

	e := newEmbedderWithTransport("http://ollama", "model", rt)
	vec, err := e.CreateEmbedding(context.Background(), "single")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vec) != 1 {
		t.Fatalf("expected vector length 1, got %d", len(vec))
	}
	if vec[0] != float32(0.99) {
		t.Errorf("expected vec[0] = %f, got %f", float32(0.99), vec[0])
	}
}