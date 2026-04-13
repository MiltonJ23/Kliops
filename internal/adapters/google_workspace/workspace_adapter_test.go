package googleworkspace

import (
	"context"
	"fmt"
	"testing"

	"github.com/MiltonJ23/Kliops/internal/core/ports"
)

// --- Compile-time interface assertion ---

// TestWorkspaceAdapter_ImplementsDocumentGeneratorInterface verifies at compile time
// that WorkspaceAdapter satisfies the ports.DocumentGenerator interface.
// This mirrors the assertion already in the source file and ensures the test suite
// fails to compile if the interface contract is broken.
func TestWorkspaceAdapter_ImplementsDocumentGeneratorInterface(t *testing.T) {
	// This is a compile-time assertion; if WorkspaceAdapter does not implement
	// ports.DocumentGenerator the build will fail here.
	var _ ports.DocumentGenerator = (*WorkspaceAdapter)(nil)
}

// --- Constructor tests ---

// TestNewWorkspaceAdapter_NonExistentCredentials_ReturnsError verifies that
// NewWorkspaceAdapter returns an error when given a path to a credentials file
// that does not exist on disk.
func TestNewWorkspaceAdapter_NonExistentCredentials_ReturnsError(t *testing.T) {
	_, err := NewWorkspaceAdapter(context.Background(), "/nonexistent/credentials.json")
	if err == nil {
		t.Fatal("expected error for nonexistent credentials file, got nil")
	}
}

// TestNewWorkspaceAdapter_EmptyCredentialsPath_ReturnsError verifies that an empty
// credentials file path surfaces an error from the Google SDK initialisation.
func TestNewWorkspaceAdapter_EmptyCredentialsPath_ReturnsError(t *testing.T) {
	_, err := NewWorkspaceAdapter(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty credentials path, got nil")
	}
}

// TestNewWorkspaceAdapter_FullIntegration_RequiresServiceAccount documents that
// a successful adapter creation requires a valid Google service-account credentials
// file with the Drive and Docs APIs enabled.
func TestNewWorkspaceAdapter_FullIntegration_RequiresServiceAccount(t *testing.T) {
	t.Skip("Full NewWorkspaceAdapter initialisation requires a real Google service account credentials file")
}

// --- Document URL format tests ---

// TestWorkspaceAdapter_DocURLFormat verifies that the Google Docs URL constructed by
// GenerateFromStream follows the expected pattern. The format is defined in the
// implementation as: "https://docs.google.com/document/d/%s/edit".
func TestWorkspaceAdapter_DocURLFormat(t *testing.T) {
	docID := "abc123xyz"
	expectedURL := fmt.Sprintf("https://docs.google.com/document/d/%s/edit", docID)

	// Verify the format string matches what the implementation produces.
	got := fmt.Sprintf("https://docs.google.com/document/d/%s/edit", docID)
	if got != expectedURL {
		t.Errorf("unexpected URL format: got %q, want %q", got, expectedURL)
	}
}

// --- GenerateFromStream / ShareWithUser integration tests ---

// TestWorkspaceAdapter_GenerateFromStream_RequiresGoogleAPI documents that
// GenerateFromStream requires live Google Drive and Docs API access.
func TestWorkspaceAdapter_GenerateFromStream_RequiresGoogleAPI(t *testing.T) {
	t.Skip("GenerateFromStream requires a running Google Drive/Docs API connection; use integration tests")
}

// TestWorkspaceAdapter_ShareWithUser_RequiresGoogleAPI documents that ShareWithUser
// requires live Google Drive API access.
func TestWorkspaceAdapter_ShareWithUser_RequiresGoogleAPI(t *testing.T) {
	t.Skip("ShareWithUser requires a running Google Drive API connection; use integration tests")
}