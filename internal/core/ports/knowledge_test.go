package ports

import (
	"context"
	"testing"

	"github.com/MiltonJ23/Kliops/internal/core/domain"
)

// Compile-time checks: verify that if a type implements the interfaces,
// the interface methods are correctly satisfied.
// These will fail to compile if the interface definitions change incompatibly.

// mockEmbedder is a minimal implementation used to verify the Embedder interface.
type mockEmbedder struct {
	result []float32
	err    error
}

func (m *mockEmbedder) CreateEmbedding(_ context.Context, _ string) ([]float32, error) {
	return m.result, m.err
}

// mockKnowledgeBase is a minimal implementation used to verify the KnowledgeBase interface.
type mockKnowledgeBase struct {
	ingestErr     error
	searchResults []SearchResult
	searchErr     error
}

func (m *mockKnowledgeBase) Ingest(_ context.Context, _ domain.ReponseHistorique) error {
	return m.ingestErr
}

func (m *mockKnowledgeBase) SearchSimilar(_ context.Context, _ string, _ int) ([]SearchResult, error) {
	return m.searchResults, m.searchErr
}

// Compile-time interface assertions.
var _ Embedder = (*mockEmbedder)(nil)
var _ KnowledgeBase = (*mockKnowledgeBase)(nil)

func TestSearchResultEmbedding(t *testing.T) {
	// SearchResult embeds domain.ReponseHistorique — all its fields must be accessible directly.
	r := SearchResult{
		ReponseHistorique: domain.ReponseHistorique{
			ID:                "rh-001",
			AppelOffreID:      "AO-001",
			ExigenceTechnique: "Pose de carrelage",
			ReponseApportee:   "Carrelage grès cérame",
			PrixPropose:       50000.0,
			Gagne:             true,
		},
		SimilarityScore: 0.95,
	}

	if r.ID != "rh-001" {
		t.Errorf("expected embedded ID %q, got %q", "rh-001", r.ID)
	}
	if r.AppelOffreID != "AO-001" {
		t.Errorf("expected embedded AppelOffreID %q, got %q", "AO-001", r.AppelOffreID)
	}
	if r.ExigenceTechnique != "Pose de carrelage" {
		t.Errorf("expected embedded ExigenceTechnique %q, got %q", "Pose de carrelage", r.ExigenceTechnique)
	}
	if r.ReponseApportee != "Carrelage grès cérame" {
		t.Errorf("expected embedded ReponseApportee %q, got %q", "Carrelage grès cérame", r.ReponseApportee)
	}
	if r.PrixPropose != 50000.0 {
		t.Errorf("expected embedded PrixPropose %f, got %f", 50000.0, r.PrixPropose)
	}
	if !r.Gagne {
		t.Errorf("expected embedded Gagne true, got false")
	}
	if r.SimilarityScore != 0.95 {
		t.Errorf("expected SimilarityScore 0.95, got %f", r.SimilarityScore)
	}
}

func TestSearchResultZeroSimilarityScore(t *testing.T) {
	r := SearchResult{}
	if r.SimilarityScore != 0.0 {
		t.Errorf("expected zero SimilarityScore, got %f", r.SimilarityScore)
	}
}

func TestSearchResultPerfectScore(t *testing.T) {
	r := SearchResult{SimilarityScore: 1.0}
	if r.SimilarityScore != 1.0 {
		t.Errorf("expected SimilarityScore 1.0, got %f", r.SimilarityScore)
	}
}

func TestSearchResultLowScore(t *testing.T) {
	r := SearchResult{SimilarityScore: 0.1}
	if r.SimilarityScore != 0.1 {
		t.Errorf("expected SimilarityScore 0.1, got %f", r.SimilarityScore)
	}
}

func TestEmbedderInterfaceReturnVector(t *testing.T) {
	expected := []float32{0.1, 0.2, 0.3}
	e := &mockEmbedder{result: expected}

	got, err := e.CreateEmbedding(context.Background(), "test text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("expected vector length %d, got %d", len(expected), len(got))
	}
	for i, v := range expected {
		if got[i] != v {
			t.Errorf("vector[%d]: expected %f, got %f", i, v, got[i])
		}
	}
}

func TestKnowledgeBaseIngestDelegates(t *testing.T) {
	kb := &mockKnowledgeBase{}
	err := kb.Ingest(context.Background(), domain.ReponseHistorique{ID: "id-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestKnowledgeBaseSearchSimilarDelegates(t *testing.T) {
	expected := []SearchResult{
		{SimilarityScore: 0.8},
	}
	kb := &mockKnowledgeBase{searchResults: expected}

	results, err := kb.SearchSimilar(context.Background(), "query", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].SimilarityScore != 0.8 {
		t.Errorf("expected SimilarityScore 0.8, got %f", results[0].SimilarityScore)
	}
}