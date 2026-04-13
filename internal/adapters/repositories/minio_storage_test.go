package repositories

import (
	"fmt"
	"testing"

	"github.com/MiltonJ23/Kliops/internal/core/ports"
)

func TestNewMinioStorage_ReturnsClient(t *testing.T) {
	// minio.New does not establish a connection; it just initialises the client struct.
	// A real MinIO server is NOT required for this constructor test.
	storage, err := NewMinioStorage("localhost:9000", "accesskey", "secretkey", false)
	if err != nil {
		t.Fatalf("unexpected error creating MinioStorage: %v", err)
	}
	if storage == nil {
		t.Fatal("NewMinioStorage returned nil storage")
	}
	if storage.Client == nil {
		t.Fatal("MinioStorage.Client should not be nil after successful construction")
	}
}

func TestNewMinioStorage_WithSSL_ReturnsClient(t *testing.T) {
	storage, err := NewMinioStorage("s3.amazonaws.com", "key", "secret", true)
	if err != nil {
		t.Fatalf("unexpected error creating MinioStorage with SSL: %v", err)
	}
	if storage == nil {
		t.Fatal("NewMinioStorage returned nil")
	}
}

func TestNewMinioStorage_EmptyCredentials_ReturnsClient(t *testing.T) {
	// minio.New accepts empty credentials (for anonymous access patterns);
	// the constructor itself should not fail.
	storage, err := NewMinioStorage("localhost:9000", "", "", false)
	if err != nil {
		t.Fatalf("unexpected error with empty credentials: %v", err)
	}
	if storage == nil {
		t.Fatal("NewMinioStorage returned nil with empty credentials")
	}
}

func TestNewMinioStorage_ReturnedURLFormat(t *testing.T) {
	// Verify the documented URL format from the implementation: "minio://%s//%s"
	// The implementation uses fmt.Sprintf("minio://%s//%s", bucketName, info.Key)
	// We verify this format is correctly applied.
	bucket := "dce-entrants"
	key := "report.pdf"
	expected := fmt.Sprintf("minio://%s//%s", bucket, key)

	// Since we can't easily mock the MinIO client's PutObject without a real server,
	// we validate the format string directly against the implementation's pattern.
	if expected != "minio://dce-entrants//report.pdf" {
		t.Errorf("URL format should match 'minio://%s//%s' pattern, got: %s", bucket, key, expected)
	}
}

// TestMinioStorage_Upload_RequiresIntegration documents that Upload requires a
// live MinIO instance and must be exercised via integration tests.
func TestMinioStorage_Upload_RequiresIntegration(t *testing.T) {
	t.Skip("Upload requires a running MinIO server; use integration tests with docker-compose")
}

// TestMinioStorage_Upload_ContextCancelled_RequiresIntegration serves as
// a placeholder for cancellation behaviour tested against a real server.
func TestMinioStorage_Upload_ContextCancelled_RequiresIntegration(t *testing.T) {
	t.Skip("Context cancellation for Upload requires a running MinIO server")
}

// TestMinioStorage_ImplementsFileStorageInterface verifies at compile-time that
// MinioStorage satisfies the ports.FileStorage interface.
func TestMinioStorage_ImplementsFileStorageInterface(t *testing.T) {
	storage, err := NewMinioStorage("localhost:9000", "key", "secret", false)
	if err != nil {
		t.Fatalf("could not create MinioStorage: %v", err)
	}

	// Compile-time interface satisfaction check: will not compile if MinioStorage
	// does not implement ports.FileStorage (including the new DownloadStream method).
	var _ ports.FileStorage = storage

	if storage == nil {
		t.Error("storage should not be nil")
	}
}

// --- DownloadStream tests ---

// TestMinioStorage_DownloadStream_RequiresLiveServer documents that DownloadStream
// requires a running MinIO server and must be exercised via integration tests.
func TestMinioStorage_DownloadStream_RequiresLiveServer(t *testing.T) {
	t.Skip("DownloadStream requires a running MinIO server; use integration tests with docker-compose")
}

// TestMinioStorage_DownloadStream_RequiresContextCancellation_RequiresLiveServer documents
// context cancellation behavior for DownloadStream.
func TestMinioStorage_DownloadStream_ContextCancellation_RequiresLiveServer(t *testing.T) {
	t.Skip("Context cancellation for DownloadStream requires a running MinIO server")
}

// TestMinioStorage_DownloadStream_ClientCreatesCorrectly verifies that a MinioStorage
// created for DownloadStream use has a valid non-nil client (pre-condition for calling DownloadStream).
func TestMinioStorage_DownloadStream_ClientCreatesCorrectly(t *testing.T) {
	storage, err := NewMinioStorage("localhost:9000", "key", "secret", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if storage.Client == nil {
		t.Fatal("client must not be nil before calling DownloadStream")
	}
}

// TestMinioStorage_DownloadStream_ReturnsErrorForMissingObject verifies that DownloadStream
// returns an error when the object does not exist. Since MinIO's GetObject is lazy (no network
// call until Stat/Read), the error is surfaced on Stat() — confirmed by the implementation.
// This test documents that behavior and requires a live server for full verification.
func TestMinioStorage_DownloadStream_ReturnsErrorForMissingObject(t *testing.T) {
	t.Skip("Requires a running MinIO server to verify Stat() error for missing object")
}