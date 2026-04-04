package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MiltonJ23/Kliops/internal/core/services"
)

// mockFileStorage implements ports.FileStorage for testing.
type mockFileStorage struct {
	uploadPath string
	uploadErr  error
}

func (m *mockFileStorage) Upload(_ context.Context, _, _ string, _ io.Reader, _ int64, _ string) (string, error) {
	return m.uploadPath, m.uploadErr
}

// mockPricingStrategyForHandler implements ports.PricingStrategy for handler tests.
type mockPricingStrategyForHandler struct {
	price float64
	err   error
}

func (m *mockPricingStrategyForHandler) GetPrice(_ context.Context, _ string) (float64, error) {
	return m.price, m.err
}

func newTestGateway(storage *mockFileStorage, price float64, priceErr error, source string) *GatewayHandler {
	pricingSvc := services.NewPricingService()
	pricingSvc.RegisterStrategy(source, &mockPricingStrategyForHandler{price: price, err: priceErr})
	return NewGatewayHandler(storage, pricingSvc)
}

// buildMultipartRequest builds a POST request with a multipart form containing a "document" file.
func buildMultipartRequest(t *testing.T, fieldName, filename, content string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err = io.WriteString(part, content); err != nil {
		t.Fatalf("failed to write form content: %v", err)
	}
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

// --- HandleUpload tests ---

func TestHandleUpload_Success(t *testing.T) {
	storage := &mockFileStorage{uploadPath: "minio://dce-entrants//test.pdf"}
	handler := newTestGateway(storage, 0, nil, "excel")

	req := buildMultipartRequest(t, "document", "test.pdf", "fake pdf content")
	rr := httptest.NewRecorder()

	handler.HandleUpload(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if resp["status"] != "success" {
		t.Errorf("expected status 'success', got %q", resp["status"])
	}
	if resp["path"] != "minio://dce-entrants//test.pdf" {
		t.Errorf("unexpected path: %q", resp["path"])
	}
}

func TestHandleUpload_MissingDocumentField_Returns400(t *testing.T) {
	storage := &mockFileStorage{}
	handler := newTestGateway(storage, 0, nil, "excel")

	// form with wrong field name
	req := buildMultipartRequest(t, "wrong_field", "file.pdf", "content")
	rr := httptest.NewRecorder()

	handler.HandleUpload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing 'document' field, got %d", rr.Code)
	}
}

func TestHandleUpload_InvalidMultipartBody_Returns400(t *testing.T) {
	storage := &mockFileStorage{}
	handler := newTestGateway(storage, 0, nil, "excel")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload", strings.NewReader("not a multipart body"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=INVALID")
	rr := httptest.NewRecorder()

	handler.HandleUpload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid multipart body, got %d", rr.Code)
	}
}

func TestHandleUpload_StorageError_Returns500(t *testing.T) {
	storage := &mockFileStorage{uploadErr: errors.New("minio unavailable")}
	handler := newTestGateway(storage, 0, nil, "excel")

	req := buildMultipartRequest(t, "document", "file.pdf", "content")
	rr := httptest.NewRecorder()

	handler.HandleUpload(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when storage returns error, got %d", rr.Code)
	}
}

func TestHandleUpload_ResponseContainsPath(t *testing.T) {
	expectedPath := "minio://dce-entrants//report.docx"
	storage := &mockFileStorage{uploadPath: expectedPath}
	handler := newTestGateway(storage, 0, nil, "excel")

	req := buildMultipartRequest(t, "document", "report.docx", "doc content")
	rr := httptest.NewRecorder()
	handler.HandleUpload(rr, req)

	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["path"] != expectedPath {
		t.Errorf("expected path %q in response, got %q", expectedPath, resp["path"])
	}
}

// --- HandlePrice tests ---

func TestHandlePrice_Success(t *testing.T) {
	storage := &mockFileStorage{}
	handler := newTestGateway(storage, 150.50, nil, "excel")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/price?source=excel&code=ART01", nil)
	rr := httptest.NewRecorder()

	handler.HandlePrice(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if resp["code_article"] != "ART01" {
		t.Errorf("expected code_article 'ART01', got %v", resp["code_article"])
	}
	if resp["source"] != "excel" {
		t.Errorf("expected source 'excel', got %v", resp["source"])
	}
	if resp["prix"] != 150.50 {
		t.Errorf("expected prix 150.50, got %v", resp["prix"])
	}
}

func TestHandlePrice_MissingSource_Returns400(t *testing.T) {
	handler := newTestGateway(&mockFileStorage{}, 0, nil, "excel")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/price?code=ART01", nil)
	rr := httptest.NewRecorder()

	handler.HandlePrice(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when 'source' is missing, got %d", rr.Code)
	}
}

func TestHandlePrice_MissingCode_Returns400(t *testing.T) {
	handler := newTestGateway(&mockFileStorage{}, 0, nil, "excel")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/price?source=excel", nil)
	rr := httptest.NewRecorder()

	handler.HandlePrice(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when 'code' is missing, got %d", rr.Code)
	}
}

func TestHandlePrice_BothParamsMissing_Returns400(t *testing.T) {
	handler := newTestGateway(&mockFileStorage{}, 0, nil, "excel")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/price", nil)
	rr := httptest.NewRecorder()

	handler.HandlePrice(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when both params are missing, got %d", rr.Code)
	}
}

func TestHandlePrice_UnknownSource_Returns404(t *testing.T) {
	handler := newTestGateway(&mockFileStorage{}, 0, nil, "excel")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/price?source=unknown&code=ART01", nil)
	rr := httptest.NewRecorder()

	handler.HandlePrice(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown strategy, got %d", rr.Code)
	}
}

func TestHandlePrice_PricingError_Returns404(t *testing.T) {
	storage := &mockFileStorage{}
	handler := newTestGateway(storage, 0, errors.New("article not found"), "excel")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/price?source=excel&code=NOTEXIST", nil)
	rr := httptest.NewRecorder()

	handler.HandlePrice(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 when pricing returns error, got %d", rr.Code)
	}
}

func TestHandlePrice_ResponseJSON_ContainsAllFields(t *testing.T) {
	handler := newTestGateway(&mockFileStorage{}, 42.0, nil, "postgres")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/price?source=postgres&code=MAT99", nil)
	rr := httptest.NewRecorder()
	handler.HandlePrice(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)

	for _, key := range []string{"code_article", "prix", "source"} {
		if _, ok := resp[key]; !ok {
			t.Errorf("expected key %q in response JSON", key)
		}
	}
}

// --- NewGatewayHandler tests ---

func TestNewGatewayHandler_SetsFields(t *testing.T) {
	storage := &mockFileStorage{}
	pricingSvc := services.NewPricingService()

	h := NewGatewayHandler(storage, pricingSvc)

	if h == nil {
		t.Fatal("NewGatewayHandler returned nil")
	}
	if h.Storage != storage {
		t.Error("Storage field not set correctly")
	}
	if h.Pricing != pricingSvc {
		t.Error("Pricing field not set correctly")
	}
}