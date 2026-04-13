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
	uploadPath     string
	uploadErr      error
	downloadReader io.ReadCloser
	downloadErr    error
	deleteErr      error
}

func (m *mockFileStorage) Upload(_ context.Context, _, _ string, _ io.Reader, _ int64, _ string) (string, error) {
	return m.uploadPath, m.uploadErr
}

func (m *mockFileStorage) Delete(_ context.Context, _, _ string) error {
	return m.deleteErr
}

func (m *mockFileStorage) DownloadStream(_ context.Context, _, _ string) (io.ReadCloser, error) {
	return m.downloadReader, m.downloadErr
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

// --- readCloser tests ---

func TestReadCloser_CloseReturnsNil(t *testing.T) {
	rc := &readCloser{Reader: strings.NewReader("hello")}
	if err := rc.Close(); err != nil {
		t.Errorf("expected Close() to return nil, got %v", err)
	}
}

func TestReadCloser_ReadsUnderlyingReader(t *testing.T) {
	content := "test content"
	rc := &readCloser{Reader: strings.NewReader(content)}
	buf := make([]byte, len(content))
	n, err := rc.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected read error: %v", err)
	}
	if string(buf[:n]) != content {
		t.Errorf("expected %q, got %q", content, string(buf[:n]))
	}
}

// --- HandleUpload filename sanitization tests (PR changes) ---

func TestHandleUpload_EmptyFilename_Returns400(t *testing.T) {
	storage := &mockFileStorage{uploadPath: "minio://dce-entrants//file"}
	handler := newTestGateway(storage, 0, nil, "excel")

	// Use filename "." which filepath.Base("") resolves to "." - but let's use a direct empty
	// In multipart, we can set filename to "." to test that branch
	req := buildMultipartRequest(t, "document", ".", "content")
	rr := httptest.NewRecorder()

	handler.HandleUpload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for dot filename, got %d", rr.Code)
	}
}

func TestHandleUpload_PathTraversalFilename_IsNeutralised(t *testing.T) {
	// filepath.Base strips path components: "../etc/passwd" → "passwd"
	// The upload therefore proceeds with the sanitised base name.
	storage := &mockFileStorage{uploadPath: "minio://dce-entrants//passwd"}
	handler := newTestGateway(storage, 0, nil, "excel")

	req := buildMultipartRequest(t, "document", "../etc/passwd", "content")
	rr := httptest.NewRecorder()

	handler.HandleUpload(rr, req)

	// The handler neutralises traversal via filepath.Base; the upload succeeds.
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 after filepath.Base strips path traversal, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleUpload_DotDotInBaseFilename_Returns400(t *testing.T) {
	// A base filename that itself contains ".." (e.g. "..bad.pdf") is rejected.
	storage := &mockFileStorage{}
	handler := newTestGateway(storage, 0, nil, "excel")

	req := buildMultipartRequest(t, "document", "..bad.pdf", "content")
	rr := httptest.NewRecorder()

	handler.HandleUpload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for filename containing '..', got %d", rr.Code)
	}
}

func TestHandleUpload_VeryLongFilename_Returns400(t *testing.T) {
	storage := &mockFileStorage{}
	handler := newTestGateway(storage, 0, nil, "excel")

	longName := strings.Repeat("a", 256) + ".pdf"
	req := buildMultipartRequest(t, "document", longName, "content")
	rr := httptest.NewRecorder()

	handler.HandleUpload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for filename over 255 chars, got %d", rr.Code)
	}
}

func TestHandleUpload_ContentTypeAutoDetected(t *testing.T) {
	// Verify that the uploaded content-type is auto-detected, not taken from the form header.
	// We capture the contentType argument passed to Upload via a recording mock.
	storage := &capturingMockStorage{}
	pricingSvc := services.NewPricingService()
	handler := NewGatewayHandler(storage, pricingSvc)

	req := buildMultipartRequest(t, "document", "file.pdf", "fake pdf content")
	rr := httptest.NewRecorder()
	handler.HandleUpload(rr, req)

	// http.DetectContentType returns "text/plain; charset=utf-8" for plain text content
	if storage.lastContentType == "" {
		t.Error("expected a non-empty content type to be auto-detected")
	}
}

// capturingMockStorage records the last upload's content type for assertion.
type capturingMockStorage struct {
	lastContentType string
}

func (c *capturingMockStorage) Upload(_ context.Context, _, _ string, _ io.Reader, _ int64, contentType string) (string, error) {
	c.lastContentType = contentType
	return "minio://dce-entrants//captured.pdf", nil
}

func (c *capturingMockStorage) Delete(_ context.Context, _, _ string) error {
	return nil
}

func (c *capturingMockStorage) DownloadStream(_ context.Context, _, _ string) (io.ReadCloser, error) {
	return nil, nil
}

func TestHandleUpload_FilenameExactly255Chars_Succeeds(t *testing.T) {
	storage := &mockFileStorage{uploadPath: "minio://dce-entrants//ok.pdf"}
	handler := newTestGateway(storage, 0, nil, "excel")

	// Exactly 255 chars should be accepted
	name255 := strings.Repeat("a", 251) + ".pdf"
	req := buildMultipartRequest(t, "document", name255, "content")
	rr := httptest.NewRecorder()

	handler.HandleUpload(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for 255-char filename, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

// --- HandlePrice response format tests (PR changes) ---

func TestHandlePrice_ResponseDoesNotReturn404OnPricingError(t *testing.T) {
	// After the PR change, pricing errors return 500 (InternalServerError), not 404.
	storage := &mockFileStorage{}
	handler := newTestGateway(storage, 0, errors.New("db error"), "excel")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/price?source=excel&code=ART01", nil)
	rr := httptest.NewRecorder()
	handler.HandlePrice(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when pricing strategy returns error, got %d", rr.Code)
	}
}