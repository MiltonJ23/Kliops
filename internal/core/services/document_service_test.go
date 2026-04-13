package services

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

// mockDocumentStorage implements ports.FileStorage for DocumentService tests.
type mockDocumentStorage struct {
	downloadReader io.ReadCloser
	downloadErr    error
	uploadErr      error
	deleteErr      error
}

func (m *mockDocumentStorage) Upload(_ context.Context, _, _ string, _ io.Reader, _ int64, _ string) (string, error) {
	return "", m.uploadErr
}

func (m *mockDocumentStorage) Delete(_ context.Context, _, _ string) error {
	return m.deleteErr
}

func (m *mockDocumentStorage) DownloadStream(_ context.Context, _, _ string) (io.ReadCloser, error) {
	return m.downloadReader, m.downloadErr
}

// nopReadCloser wraps an io.Reader with a no-op Close for testing.
type nopReadCloser struct {
	io.Reader
}

func (n *nopReadCloser) Close() error { return nil }

// mockDocumentGenerator implements ports.DocumentGenerator for tests.
type mockDocumentGenerator struct {
	docID    string
	docURL   string
	genErr   error
	shareErr error
}

func (m *mockDocumentGenerator) GenerateFromStream(_ context.Context, _ io.Reader, _ string, _ map[string]string) (string, string, error) {
	return m.docID, m.docURL, m.genErr
}

func (m *mockDocumentGenerator) ShareWithUser(_ context.Context, _ string, _ string) error {
	return m.shareErr
}

// --- NewDocumentService tests ---

func TestNewDocumentService_ReturnsNonNil(t *testing.T) {
	storage := &mockDocumentStorage{}
	gen := &mockDocumentGenerator{}

	svc := NewDocumentService(storage, gen)

	if svc == nil {
		t.Fatal("NewDocumentService returned nil")
	}
}

func TestNewDocumentService_SetsStorageField(t *testing.T) {
	storage := &mockDocumentStorage{}
	gen := &mockDocumentGenerator{}

	svc := NewDocumentService(storage, gen)

	if svc.Storage != storage {
		t.Error("Storage field not set correctly")
	}
}

func TestNewDocumentService_SetsGeneratorField(t *testing.T) {
	storage := &mockDocumentStorage{}
	gen := &mockDocumentGenerator{}

	svc := NewDocumentService(storage, gen)

	if svc.Generator != gen {
		t.Error("Generator field not set correctly")
	}
}

// --- CompileTechnicalMemory tests ---

func TestCompileTechnicalMemory_Success_ReturnsDocURL(t *testing.T) {
	expectedURL := "https://docs.google.com/document/d/abc123/edit"
	storage := &mockDocumentStorage{
		downloadReader: &nopReadCloser{Reader: strings.NewReader("template content")},
	}
	gen := &mockDocumentGenerator{
		docID:  "abc123",
		docURL: expectedURL,
	}

	svc := NewDocumentService(storage, gen)
	url, err := svc.CompileTechnicalMemory(context.Background(), "Project Alpha", map[string]string{"name": "ACME"}, "user@example.com")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != expectedURL {
		t.Errorf("expected URL %q, got %q", expectedURL, url)
	}
}

func TestCompileTechnicalMemory_StorageDownloadError_ReturnsError(t *testing.T) {
	storage := &mockDocumentStorage{
		downloadErr: errors.New("minio unreachable"),
	}
	gen := &mockDocumentGenerator{}

	svc := NewDocumentService(storage, gen)
	_, err := svc.CompileTechnicalMemory(context.Background(), "Project Beta", nil, "user@example.com")

	if err == nil {
		t.Fatal("expected error when storage download fails, got nil")
	}
	if !strings.Contains(err.Error(), "failed to stream the Template") {
		t.Errorf("expected error message to mention template streaming, got: %v", err)
	}
}

func TestCompileTechnicalMemory_GenerationError_ReturnsEmptyURLAndError(t *testing.T) {
	storage := &mockDocumentStorage{
		downloadReader: &nopReadCloser{Reader: strings.NewReader("template content")},
	}
	genErr := errors.New("drive API quota exceeded")
	gen := &mockDocumentGenerator{
		genErr: genErr,
	}

	svc := NewDocumentService(storage, gen)
	url, err := svc.CompileTechnicalMemory(context.Background(), "Project Gamma", nil, "user@example.com")

	if err == nil {
		t.Fatal("expected error when generation fails, got nil")
	}
	if url != "" {
		t.Errorf("expected empty URL when generation fails, got %q", url)
	}
	if !strings.Contains(err.Error(), "document generation failed") {
		t.Errorf("expected error to mention document generation, got: %v", err)
	}
	if !errors.Is(err, genErr) {
		t.Errorf("expected wrapped generation error, got: %v", err)
	}
}

func TestCompileTechnicalMemory_SharingError_ReturnsURLWithError(t *testing.T) {
	// When sharing fails, the service should return the docURL alongside the error.
	expectedURL := "https://docs.google.com/document/d/xyz999/edit"
	storage := &mockDocumentStorage{
		downloadReader: &nopReadCloser{Reader: strings.NewReader("template content")},
	}
	shareErr := errors.New("permission denied")
	gen := &mockDocumentGenerator{
		docID:    "xyz999",
		docURL:   expectedURL,
		shareErr: shareErr,
	}

	svc := NewDocumentService(storage, gen)
	url, err := svc.CompileTechnicalMemory(context.Background(), "Project Delta", nil, "user@example.com")

	if err == nil {
		t.Fatal("expected error when sharing fails, got nil")
	}
	// URL must still be returned even though sharing failed
	if url != expectedURL {
		t.Errorf("expected URL %q even on sharing failure, got %q", expectedURL, url)
	}
	if !strings.Contains(err.Error(), "document generated but sharing failed") {
		t.Errorf("expected error to mention sharing failure, got: %v", err)
	}
	if !errors.Is(err, shareErr) {
		t.Errorf("expected wrapped sharing error, got: %v", err)
	}
}

func TestCompileTechnicalMemory_EmptyVariables_DoesNotError(t *testing.T) {
	storage := &mockDocumentStorage{
		downloadReader: &nopReadCloser{Reader: strings.NewReader("template content")},
	}
	gen := &mockDocumentGenerator{
		docID:  "doc1",
		docURL: "https://docs.google.com/document/d/doc1/edit",
	}

	svc := NewDocumentService(storage, gen)
	_, err := svc.CompileTechnicalMemory(context.Background(), "Project Empty Vars", map[string]string{}, "user@example.com")

	if err != nil {
		t.Fatalf("expected no error with empty variables map, got: %v", err)
	}
}

func TestCompileTechnicalMemory_NilVariables_DoesNotError(t *testing.T) {
	storage := &mockDocumentStorage{
		downloadReader: &nopReadCloser{Reader: strings.NewReader("template content")},
	}
	gen := &mockDocumentGenerator{
		docID:  "doc2",
		docURL: "https://docs.google.com/document/d/doc2/edit",
	}

	svc := NewDocumentService(storage, gen)
	_, err := svc.CompileTechnicalMemory(context.Background(), "Project Nil Vars", nil, "user@example.com")

	if err != nil {
		t.Fatalf("expected no error with nil variables, got: %v", err)
	}
}

func TestCompileTechnicalMemory_DocNameContainsProjectName(t *testing.T) {
	// Verify that the document name passed to GenerateFromStream contains the project name.
	var capturedDocName string
	captureGen := &capturingDocGenerator{
		onGenerate: func(name string) { capturedDocName = name },
		docID:      "captured123",
		docURL:     "https://docs.google.com/document/d/captured123/edit",
	}
	storage := &mockDocumentStorage{
		downloadReader: &nopReadCloser{Reader: strings.NewReader("template content")},
	}

	svc := NewDocumentService(storage, captureGen)
	_, err := svc.CompileTechnicalMemory(context.Background(), "My Special Project", nil, "user@example.com")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedDocName, "My Special Project") {
		t.Errorf("expected doc name to contain project name 'My Special Project', got %q", capturedDocName)
	}
}

// capturingDocGenerator records arguments passed to GenerateFromStream.
type capturingDocGenerator struct {
	onGenerate func(fileName string)
	docID      string
	docURL     string
}

func (c *capturingDocGenerator) GenerateFromStream(_ context.Context, _ io.Reader, fileName string, _ map[string]string) (string, string, error) {
	if c.onGenerate != nil {
		c.onGenerate(fileName)
	}
	return c.docID, c.docURL, nil
}

func (c *capturingDocGenerator) ShareWithUser(_ context.Context, _ string, _ string) error {
	return nil
}

// TestCompileTechnicalMemory_ClosesTemplateStream verifies the template stream is closed after use.
func TestCompileTechnicalMemory_ClosesTemplateStream(t *testing.T) {
	closeCalled := false
	trackingReader := &trackingReadCloser{
		Reader: strings.NewReader("template"),
		onClose: func() { closeCalled = true },
	}
	storage := &mockDocumentStorage{downloadReader: trackingReader}
	gen := &mockDocumentGenerator{docID: "d1", docURL: "https://docs.google.com/document/d/d1/edit"}

	svc := NewDocumentService(storage, gen)
	svc.CompileTechnicalMemory(context.Background(), "Proj", nil, "u@example.com")

	if !closeCalled {
		t.Error("expected template stream to be closed via defer, but Close() was not called")
	}
}

// trackingReadCloser records whether Close was called.
type trackingReadCloser struct {
	io.Reader
	onClose func()
}

func (t *trackingReadCloser) Close() error {
	if t.onClose != nil {
		t.onClose()
	}
	return nil
}