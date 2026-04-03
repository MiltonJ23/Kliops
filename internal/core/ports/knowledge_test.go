package ports

import (
	"context"
	"errors"
	"testing"

	"github.com/MiltonJ23/Kliops/internal/core/domain"
)

// --- Compile-time interface checks ---

// mockKnowledgeBase is a test implementation of KnowledgeBase.
type mockKnowledgeBase struct {
	ingestCalled       bool
	searchCalled       bool
	ingestErr          error
	searchResults      []SearchResult
	searchErr          error
	lastIngested       domain.ReponseHistorique
	lastSearchQuery    string
	lastSearchLimit    int
}

func (m *mockKnowledgeBase) Ingest(ctx context.Context, reponse domain.ReponseHistorique) error {
	m.ingestCalled = true
	m.lastIngested = reponse
	return m.ingestErr
}

func (m *mockKnowledgeBase) SearchSimilar(ctx context.Context, nouvelleExigence string, limit int) ([]SearchResult, error) {
	m.searchCalled = true
	m.lastSearchQuery = nouvelleExigence
	m.lastSearchLimit = limit
	return m.searchResults, m.searchErr
}

// mockEmbedder is a test implementation of Embedder.
type mockEmbedder struct {
	embedding []float32
	err       error
	callCount int
	lastText  string
}

func (m *mockEmbedder) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
	m.callCount++
	m.lastText = text
	return m.embedding, m.err
}

// Ensure the mock types satisfy the interfaces at compile time.
var _ KnowledgeBase = (*mockKnowledgeBase)(nil)
var _ Embedder = (*mockEmbedder)(nil)

// --- SearchResult struct tests ---

func TestSearchResultEmbedding(t *testing.T) {
	rh := domain.ReponseHistorique{
		ID:              "RH-001",
		AppelOffreID:    "AO-001",
		ReponseApportee: "Réponse détaillée",
		Gagne:           true,
	}
	sr := SearchResult{
		ReponseHistorique: rh,
		SimilarityScore:   0.95,
	}

	if sr.ID != "RH-001" {
		t.Errorf("expected embedded ID RH-001, got %q", sr.ID)
	}
	if sr.AppelOffreID != "AO-001" {
		t.Errorf("expected embedded AppelOffreID AO-001, got %q", sr.AppelOffreID)
	}
	if sr.ReponseApportee != "Réponse détaillée" {
		t.Errorf("unexpected ReponseApportee: %q", sr.ReponseApportee)
	}
	if !sr.Gagne {
		t.Error("expected Gagne true via embedding")
	}
	if sr.SimilarityScore != 0.95 {
		t.Errorf("expected SimilarityScore 0.95, got %f", sr.SimilarityScore)
	}
}

func TestSearchResultSimilarityScoreBoundaries(t *testing.T) {
	tests := []struct {
		name  string
		score float32
	}{
		{"perfect match", 1.0},
		{"no match", 0.0},
		{"half match", 0.5},
		{"near perfect", 0.999},
		{"near zero", 0.001},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sr := SearchResult{SimilarityScore: tc.score}
			if sr.SimilarityScore != tc.score {
				t.Errorf("expected score %f, got %f", tc.score, sr.SimilarityScore)
			}
		})
	}
}

func TestSearchResultZeroValue(t *testing.T) {
	var sr SearchResult
	if sr.SimilarityScore != 0 {
		t.Errorf("expected zero SimilarityScore, got %f", sr.SimilarityScore)
	}
	if sr.ID != "" {
		t.Errorf("expected empty embedded ID, got %q", sr.ID)
	}
	if sr.Gagne {
		t.Error("expected Gagne false by default")
	}
}

// --- KnowledgeBase interface behaviour tests (via mock) ---

func TestKnowledgeBaseIngestDelegates(t *testing.T) {
	kb := &mockKnowledgeBase{}
	reponse := domain.ReponseHistorique{
		ID:           "RH-T1",
		AppelOffreID: "AO-T1",
	}

	err := kb.Ingest(context.Background(), reponse)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !kb.ingestCalled {
		t.Error("Ingest was not called on the mock")
	}
	if kb.lastIngested.ID != "RH-T1" {
		t.Errorf("unexpected last ingested ID: %q", kb.lastIngested.ID)
	}
}

func TestKnowledgeBaseIngestPropagatesError(t *testing.T) {
	kb := &mockKnowledgeBase{ingestErr: errors.New("storage full")}
	err := kb.Ingest(context.Background(), domain.ReponseHistorique{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "storage full" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestKnowledgeBaseSearchSimilarDelegates(t *testing.T) {
	expected := []SearchResult{
		{
			ReponseHistorique: domain.ReponseHistorique{ReponseApportee: "Résultat 1"},
			SimilarityScore:   0.88,
		},
	}
	kb := &mockKnowledgeBase{searchResults: expected}

	results, err := kb.SearchSimilar(context.Background(), "query text", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !kb.searchCalled {
		t.Error("SearchSimilar was not called on the mock")
	}
	if kb.lastSearchQuery != "query text" {
		t.Errorf("unexpected query: %q", kb.lastSearchQuery)
	}
	if kb.lastSearchLimit != 3 {
		t.Errorf("unexpected limit: %d", kb.lastSearchLimit)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].SimilarityScore != 0.88 {
		t.Errorf("unexpected score: %f", results[0].SimilarityScore)
	}
}

func TestKnowledgeBaseSearchSimilarPropagatesError(t *testing.T) {
	kb := &mockKnowledgeBase{searchErr: errors.New("qdrant unreachable")}
	results, err := kb.SearchSimilar(context.Background(), "anything", 5)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if results != nil {
		t.Errorf("expected nil results on error, got %v", results)
	}
}

// --- Embedder interface behaviour tests (via mock) ---

func TestEmbedderCreateEmbeddingReturnsVector(t *testing.T) {
	vec := []float32{0.1, 0.2, 0.3}
	emb := &mockEmbedder{embedding: vec}

	result, err := emb.CreateEmbedding(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != len(vec) {
		t.Fatalf("expected vector length %d, got %d", len(vec), len(result))
	}
	for i, v := range result {
		if v != vec[i] {
			t.Errorf("vector[%d]: expected %f, got %f", i, vec[i], v)
		}
	}
	if emb.lastText != "hello world" {
		t.Errorf("unexpected text passed to embedder: %q", emb.lastText)
	}
}

func TestEmbedderCreateEmbeddingPropagatesError(t *testing.T) {
	emb := &mockEmbedder{err: errors.New("model not loaded")}
	result, err := emb.CreateEmbedding(context.Background(), "some text")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result on error, got %v", result)
	}
}

func TestEmbedderCreateEmbeddingEmptyText(t *testing.T) {
	vec := []float32{0.0, 0.0, 0.0}
	emb := &mockEmbedder{embedding: vec}

	result, err := emb.CreateEmbedding(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if emb.lastText != "" {
		t.Errorf("expected empty string passed, got %q", emb.lastText)
	}
	if len(result) != 3 {
		t.Errorf("expected vector length 3, got %d", len(result))
	}
}