package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// mockRoundTripper implements http.RoundTripper to intercept HTTP calls without
// requiring a real network connection or listening server.
type mockRoundTripper struct {
	response *http.Response
	err      error
	// capturedReq is set to the last request received.
	capturedReq *http.Request
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.capturedReq = req
	if req.Context().Err() != nil {
		return nil, req.Context().Err()
	}
	return m.response, m.err
}

// jsonBody creates an io.ReadCloser with JSON-encoded content.
func jsonBody(v interface{}) io.ReadCloser {
	pr, pw := io.Pipe()
	go func() {
		json.NewEncoder(pw).Encode(v)
		pw.Close()
	}()
	return pr
}

// stringBody creates an io.ReadCloser with string content.
func stringBody(s string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(s))
}

func TestNewERPPricing_SetsFields(t *testing.T) {
	erp := NewERPPricing("http://erp.example.com")
	if erp == nil {
		t.Fatal("NewERPPricing returned nil")
	}
	if erp.ApiUrl != "http://erp.example.com" {
		t.Errorf("expected ApiUrl 'http://erp.example.com', got %q", erp.ApiUrl)
	}
	if erp.Client == nil {
		t.Fatal("http.Client should be initialized")
	}
	if erp.Client.Timeout != 5*time.Second {
		t.Errorf("expected 5s timeout, got %v", erp.Client.Timeout)
	}
}

func TestERPPricing_GetPrice_Success(t *testing.T) {
	transport := &mockRoundTripper{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(map[string]float64{"prix": 150.5}),
			Header:     make(http.Header),
		},
	}

	erp := NewERPPricing("http://erp.example.com")
	erp.Client.Transport = transport

	price, err := erp.GetPrice(context.Background(), "ART01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price != 150.5 {
		t.Errorf("expected 150.5, got %f", price)
	}
}

func TestERPPricing_GetPrice_ServerReturnsNon200_ReturnsError(t *testing.T) {
	transport := &mockRoundTripper{
		response: &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       stringBody("not found"),
			Header:     make(http.Header),
		},
	}

	erp := NewERPPricing("http://erp.example.com")
	erp.Client.Transport = transport

	_, err := erp.GetPrice(context.Background(), "UNKNOWN")
	if err == nil {
		t.Fatal("expected error for non-200 response, got nil")
	}
	if !strings.Contains(err.Error(), "ERP returned status") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestERPPricing_GetPrice_ServerReturnsInvalidJSON_ReturnsError(t *testing.T) {
	transport := &mockRoundTripper{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       stringBody("not valid json {{{"),
			Header:     make(http.Header),
		},
	}

	erp := NewERPPricing("http://erp.example.com")
	erp.Client.Transport = transport

	_, err := erp.GetPrice(context.Background(), "ART01")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "unable to parse the response json") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestERPPricing_GetPrice_TransportError_ReturnsError(t *testing.T) {
	transport := &mockRoundTripper{
		err: fmt.Errorf("connection refused"),
	}

	erp := NewERPPricing("http://erp.example.com")
	erp.Client.Transport = transport

	_, err := erp.GetPrice(context.Background(), "ART01")
	if err == nil {
		t.Fatal("expected error for transport failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to call ERP") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestERPPricing_GetPrice_ContextCancelled_ReturnsError(t *testing.T) {
	transport := &mockRoundTripper{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(map[string]float64{"prix": 99.0}),
			Header:     make(http.Header),
		},
	}

	erp := NewERPPricing("http://erp.example.com")
	erp.Client.Transport = transport

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := erp.GetPrice(ctx, "ART01")
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got: %v", err)
	}
}

func TestERPPricing_GetPrice_ZeroPriceIsValid(t *testing.T) {
	transport := &mockRoundTripper{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(map[string]float64{"prix": 0.0}),
			Header:     make(http.Header),
		},
	}

	erp := NewERPPricing("http://erp.example.com")
	erp.Client.Transport = transport

	price, err := erp.GetPrice(context.Background(), "FREE01")
	if err != nil {
		t.Fatalf("unexpected error for zero price: %v", err)
	}
	if price != 0.0 {
		t.Errorf("expected 0.0, got %f", price)
	}
}

func TestERPPricing_GetPrice_RequestPathContainsCode(t *testing.T) {
	transport := &mockRoundTripper{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(map[string]float64{"prix": 99.0}),
			Header:     make(http.Header),
		},
	}

	erp := NewERPPricing("http://erp.example.com")
	erp.Client.Transport = transport

	_, err := erp.GetPrice(context.Background(), "SPECIAL-CODE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if transport.capturedReq == nil {
		t.Fatal("no request was captured")
	}
	expectedPath := "/articles/SPECIAL-CODE/prix"
	if transport.capturedReq.URL.Path != expectedPath {
		t.Errorf("expected request path %q, got %q", expectedPath, transport.capturedReq.URL.Path)
	}
}

func TestERPPricing_GetPrice_ServerReturns500_ReturnsError(t *testing.T) {
	transport := &mockRoundTripper{
		response: &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       stringBody("internal server error"),
			Header:     make(http.Header),
		},
	}

	erp := NewERPPricing("http://erp.example.com")
	erp.Client.Transport = transport

	_, err := erp.GetPrice(context.Background(), "ART01")
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

func TestERPPricing_GetPrice_RequestUsesGETMethod(t *testing.T) {
	transport := &mockRoundTripper{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(map[string]float64{"prix": 42.0}),
			Header:     make(http.Header),
		},
	}

	erp := NewERPPricing("http://erp.example.com")
	erp.Client.Transport = transport

	_, err := erp.GetPrice(context.Background(), "ART01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if transport.capturedReq == nil {
		t.Fatal("no request was captured")
	}
	if transport.capturedReq.Method != http.MethodGet {
		t.Errorf("expected GET method, got %q", transport.capturedReq.Method)
	}
}