package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/MiltonJ23/Kliops/internal/core/domain"
	"github.com/MiltonJ23/Kliops/internal/core/ports"
	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
)

// ---------------------------------------------------------------------------
// Mock: ports.Embedder
// ---------------------------------------------------------------------------

type mockEmbedder struct {
	vector []float32
	err    error
}

func (m *mockEmbedder) CreateEmbedding(_ context.Context, _ string) ([]float32, error) {
	return m.vector, m.err
}

// ---------------------------------------------------------------------------
// Mock: qdrant.PointsClient (27 methods required by the interface)
// ---------------------------------------------------------------------------

type mockPointsClient struct {
	upsertResponse *qdrant.PointsOperationResponse
	upsertErr      error
	searchResponse *qdrant.SearchResponse
	searchErr      error
}

func (m *mockPointsClient) Upsert(_ context.Context, _ *qdrant.UpsertPoints, _ ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	return m.upsertResponse, m.upsertErr
}

func (m *mockPointsClient) Search(_ context.Context, _ *qdrant.SearchPoints, _ ...grpc.CallOption) (*qdrant.SearchResponse, error) {
	return m.searchResponse, m.searchErr
}

// The remaining interface methods are required but not exercised by our code paths.
func (m *mockPointsClient) Delete(_ context.Context, _ *qdrant.DeletePoints, _ ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	return nil, nil
}
func (m *mockPointsClient) Get(_ context.Context, _ *qdrant.GetPoints, _ ...grpc.CallOption) (*qdrant.GetResponse, error) {
	return nil, nil
}
func (m *mockPointsClient) UpdateVectors(_ context.Context, _ *qdrant.UpdatePointVectors, _ ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	return nil, nil
}
func (m *mockPointsClient) DeleteVectors(_ context.Context, _ *qdrant.DeletePointVectors, _ ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	return nil, nil
}
func (m *mockPointsClient) SetPayload(_ context.Context, _ *qdrant.SetPayloadPoints, _ ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	return nil, nil
}
func (m *mockPointsClient) OverwritePayload(_ context.Context, _ *qdrant.SetPayloadPoints, _ ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	return nil, nil
}
func (m *mockPointsClient) DeletePayload(_ context.Context, _ *qdrant.DeletePayloadPoints, _ ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	return nil, nil
}
func (m *mockPointsClient) ClearPayload(_ context.Context, _ *qdrant.ClearPayloadPoints, _ ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	return nil, nil
}
func (m *mockPointsClient) CreateFieldIndex(_ context.Context, _ *qdrant.CreateFieldIndexCollection, _ ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	return nil, nil
}
func (m *mockPointsClient) DeleteFieldIndex(_ context.Context, _ *qdrant.DeleteFieldIndexCollection, _ ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	return nil, nil
}
func (m *mockPointsClient) SearchBatch(_ context.Context, _ *qdrant.SearchBatchPoints, _ ...grpc.CallOption) (*qdrant.SearchBatchResponse, error) {
	return nil, nil
}
func (m *mockPointsClient) SearchGroups(_ context.Context, _ *qdrant.SearchPointGroups, _ ...grpc.CallOption) (*qdrant.SearchGroupsResponse, error) {
	return nil, nil
}
func (m *mockPointsClient) Scroll(_ context.Context, _ *qdrant.ScrollPoints, _ ...grpc.CallOption) (*qdrant.ScrollResponse, error) {
	return nil, nil
}
func (m *mockPointsClient) Recommend(_ context.Context, _ *qdrant.RecommendPoints, _ ...grpc.CallOption) (*qdrant.RecommendResponse, error) {
	return nil, nil
}
func (m *mockPointsClient) RecommendBatch(_ context.Context, _ *qdrant.RecommendBatchPoints, _ ...grpc.CallOption) (*qdrant.RecommendBatchResponse, error) {
	return nil, nil
}
func (m *mockPointsClient) RecommendGroups(_ context.Context, _ *qdrant.RecommendPointGroups, _ ...grpc.CallOption) (*qdrant.RecommendGroupsResponse, error) {
	return nil, nil
}
func (m *mockPointsClient) Discover(_ context.Context, _ *qdrant.DiscoverPoints, _ ...grpc.CallOption) (*qdrant.DiscoverResponse, error) {
	return nil, nil
}
func (m *mockPointsClient) DiscoverBatch(_ context.Context, _ *qdrant.DiscoverBatchPoints, _ ...grpc.CallOption) (*qdrant.DiscoverBatchResponse, error) {
	return nil, nil
}
func (m *mockPointsClient) Count(_ context.Context, _ *qdrant.CountPoints, _ ...grpc.CallOption) (*qdrant.CountResponse, error) {
	return nil, nil
}
func (m *mockPointsClient) UpdateBatch(_ context.Context, _ *qdrant.UpdateBatchPoints, _ ...grpc.CallOption) (*qdrant.UpdateBatchResponse, error) {
	return nil, nil
}
func (m *mockPointsClient) Query(_ context.Context, _ *qdrant.QueryPoints, _ ...grpc.CallOption) (*qdrant.QueryResponse, error) {
	return nil, nil
}
func (m *mockPointsClient) QueryBatch(_ context.Context, _ *qdrant.QueryBatchPoints, _ ...grpc.CallOption) (*qdrant.QueryBatchResponse, error) {
	return nil, nil
}
func (m *mockPointsClient) QueryGroups(_ context.Context, _ *qdrant.QueryPointGroups, _ ...grpc.CallOption) (*qdrant.QueryGroupsResponse, error) {
	return nil, nil
}
func (m *mockPointsClient) Facet(_ context.Context, _ *qdrant.FacetCounts, _ ...grpc.CallOption) (*qdrant.FacetResponse, error) {
	return nil, nil
}
func (m *mockPointsClient) SearchMatrixPairs(_ context.Context, _ *qdrant.SearchMatrixPoints, _ ...grpc.CallOption) (*qdrant.SearchMatrixPairsResponse, error) {
	return nil, nil
}
func (m *mockPointsClient) SearchMatrixOffsets(_ context.Context, _ *qdrant.SearchMatrixPoints, _ ...grpc.CallOption) (*qdrant.SearchMatrixOffsetsResponse, error) {
	return nil, nil
}

// Compile-time check.
var _ qdrant.PointsClient = (*mockPointsClient)(nil)

// ---------------------------------------------------------------------------
// Helper: build a QdrantRepository with mock internals (bypasses gRPC dial).
// ---------------------------------------------------------------------------

func newTestRepo(embedder ports.Embedder, client qdrant.PointsClient, collection string) *QdrantRepository {
	return &QdrantRepository{
		Client:     client,
		Embedder:   embedder,
		Collection: collection,
	}
}

// ---------------------------------------------------------------------------
// QdrantRepository.Ingest tests
// ---------------------------------------------------------------------------

func TestIngest_Success(t *testing.T) {
	embedder := &mockEmbedder{vector: []float32{0.1, 0.2, 0.3}}
	client := &mockPointsClient{upsertResponse: &qdrant.PointsOperationResponse{}}

	repo := newTestRepo(embedder, client, "test_collection")

	archive := domain.ReponseHistorique{
		ID:                "uuid-001",
		AppelOffreID:      "AO-2023-001",
		ExigenceTechnique: "Méthodologie de pose de revêtements",
		ReponseApportee:   "Nous utiliserons du PVC homogène.",
		Gagne:             true,
	}

	err := repo.Ingest(context.Background(), archive)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestIngest_EmbedderError(t *testing.T) {
	embedError := errors.New("embedding service unavailable")
	embedder := &mockEmbedder{err: embedError}
	client := &mockPointsClient{}

	repo := newTestRepo(embedder, client, "col")

	err := repo.Ingest(context.Background(), domain.ReponseHistorique{ID: "uuid-002"})
	if err == nil {
		t.Fatal("expected error from embedder, got nil")
	}
	if !contains(err.Error(), "vectorization") {
		t.Errorf("expected error to mention vectorization, got: %v", err)
	}
}

func TestIngest_QdrantUpsertError(t *testing.T) {
	embedder := &mockEmbedder{vector: []float32{0.5}}
	upsertErr := errors.New("qdrant unavailable")
	client := &mockPointsClient{upsertErr: upsertErr}

	repo := newTestRepo(embedder, client, "col")

	err := repo.Ingest(context.Background(), domain.ReponseHistorique{ID: "uuid-003"})
	if err == nil {
		t.Fatal("expected error from upsert, got nil")
	}
	if !contains(err.Error(), "inserting into Qdrant") {
		t.Errorf("expected error to mention qdrant insert, got: %v", err)
	}
}

func TestIngest_GagneFieldStoredCorrectly(t *testing.T) {
	// Ensure the Gagne field (bool) is passed through without being silently dropped.
	// We verify indirectly: Ingest must not fail when Gagne is false.
	embedder := &mockEmbedder{vector: []float32{0.1}}
	client := &mockPointsClient{upsertResponse: &qdrant.PointsOperationResponse{}}
	repo := newTestRepo(embedder, client, "col")

	archive := domain.ReponseHistorique{
		ID:    "uuid-004",
		Gagne: false,
	}
	err := repo.Ingest(context.Background(), archive)
	if err != nil {
		t.Fatalf("expected no error for Gagne=false, got: %v", err)
	}
}

func TestIngest_EmptyVector(t *testing.T) {
	// Edge case: embedder returns an empty vector; Qdrant call should still be attempted.
	embedder := &mockEmbedder{vector: []float32{}}
	client := &mockPointsClient{upsertResponse: &qdrant.PointsOperationResponse{}}
	repo := newTestRepo(embedder, client, "col")

	err := repo.Ingest(context.Background(), domain.ReponseHistorique{ID: "uuid-005"})
	if err != nil {
		t.Fatalf("expected no error for empty vector, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// QdrantRepository.SearchSimilar tests
// ---------------------------------------------------------------------------

func makeSearchResponse(reponseApportee string, gagne bool, score float32) *qdrant.SearchResponse {
	return &qdrant.SearchResponse{
		Result: []*qdrant.ScoredPoint{
			{
				Score: score,
				Payload: map[string]*qdrant.Value{
					"reponse_apportee": {Kind: &qdrant.Value_StringValue{StringValue: reponseApportee}},
					"gagne":            {Kind: &qdrant.Value_BoolValue{BoolValue: gagne}},
				},
			},
		},
	}
}

func TestSearchSimilar_Success(t *testing.T) {
	embedder := &mockEmbedder{vector: []float32{0.1, 0.2}}
	searchResp := makeSearchResponse("Carrelage grès cérame", true, 0.92)
	client := &mockPointsClient{searchResponse: searchResp}

	repo := newTestRepo(embedder, client, "col")

	results, err := repo.SearchSimilar(context.Background(), "quel type de sol ?", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ReponseApportee != "Carrelage grès cérame" {
		t.Errorf("expected ReponseApportee %q, got %q", "Carrelage grès cérame", results[0].ReponseApportee)
	}
	if !results[0].Gagne {
		t.Errorf("expected Gagne true, got false")
	}
	if results[0].SimilarityScore != 0.92 {
		t.Errorf("expected SimilarityScore 0.92, got %f", results[0].SimilarityScore)
	}
}

func TestSearchSimilar_EmbedderError(t *testing.T) {
	embedder := &mockEmbedder{err: errors.New("embed failed")}
	client := &mockPointsClient{}

	repo := newTestRepo(embedder, client, "col")

	_, err := repo.SearchSimilar(context.Background(), "query", 5)
	if err == nil {
		t.Fatal("expected error from embedder, got nil")
	}
	if !contains(err.Error(), "vectorizing the request") {
		t.Errorf("expected vectorization error message, got: %v", err)
	}
}

func TestSearchSimilar_QdrantSearchError(t *testing.T) {
	embedder := &mockEmbedder{vector: []float32{0.1}}
	searchErr := errors.New("qdrant search failed")
	client := &mockPointsClient{searchErr: searchErr}

	repo := newTestRepo(embedder, client, "col")

	_, err := repo.SearchSimilar(context.Background(), "query", 3)
	if err == nil {
		t.Fatal("expected error from qdrant search, got nil")
	}
	if !contains(err.Error(), "searching for Vector Result") {
		t.Errorf("expected qdrant search error message, got: %v", err)
	}
}

func TestSearchSimilar_EmptyResults(t *testing.T) {
	embedder := &mockEmbedder{vector: []float32{0.1}}
	client := &mockPointsClient{
		searchResponse: &qdrant.SearchResponse{Result: []*qdrant.ScoredPoint{}},
	}

	repo := newTestRepo(embedder, client, "col")

	results, err := repo.SearchSimilar(context.Background(), "query", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchSimilar_MultipleResults(t *testing.T) {
	embedder := &mockEmbedder{vector: []float32{0.3, 0.7}}
	searchResp := &qdrant.SearchResponse{
		Result: []*qdrant.ScoredPoint{
			{
				Score: 0.95,
				Payload: map[string]*qdrant.Value{
					"reponse_apportee": {Kind: &qdrant.Value_StringValue{StringValue: "first response"}},
					"gagne":            {Kind: &qdrant.Value_BoolValue{BoolValue: true}},
				},
			},
			{
				Score: 0.80,
				Payload: map[string]*qdrant.Value{
					"reponse_apportee": {Kind: &qdrant.Value_StringValue{StringValue: "second response"}},
					"gagne":            {Kind: &qdrant.Value_BoolValue{BoolValue: false}},
				},
			},
		},
	}
	client := &mockPointsClient{searchResponse: searchResp}
	repo := newTestRepo(embedder, client, "col")

	results, err := repo.SearchSimilar(context.Background(), "multi query", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].ReponseApportee != "first response" {
		t.Errorf("expected first result %q, got %q", "first response", results[0].ReponseApportee)
	}
	if results[1].ReponseApportee != "second response" {
		t.Errorf("expected second result %q, got %q", "second response", results[1].ReponseApportee)
	}
	if results[0].SimilarityScore != 0.95 {
		t.Errorf("expected first score 0.95, got %f", results[0].SimilarityScore)
	}
	if results[1].SimilarityScore != 0.80 {
		t.Errorf("expected second score 0.80, got %f", results[1].SimilarityScore)
	}
}

func TestSearchSimilar_GagneFalseInResult(t *testing.T) {
	embedder := &mockEmbedder{vector: []float32{0.1}}
	searchResp := makeSearchResponse("response text", false, 0.6)
	client := &mockPointsClient{searchResponse: searchResp}
	repo := newTestRepo(embedder, client, "col")

	results, err := repo.SearchSimilar(context.Background(), "query", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Gagne {
		t.Errorf("expected Gagne false, got true")
	}
}

func TestSearchSimilar_ScoreAtBoundaries(t *testing.T) {
	embedder := &mockEmbedder{vector: []float32{1.0}}

	for _, score := range []float32{0.0, 1.0} {
		searchResp := makeSearchResponse("resp", true, score)
		client := &mockPointsClient{searchResponse: searchResp}
		repo := newTestRepo(embedder, client, "col")

		results, err := repo.SearchSimilar(context.Background(), "boundary test", 1)
		if err != nil {
			t.Fatalf("score %f: unexpected error: %v", score, err)
		}
		if results[0].SimilarityScore != score {
			t.Errorf("score %f: expected %f, got %f", score, score, results[0].SimilarityScore)
		}
	}
}

// ---------------------------------------------------------------------------
// NewQdrantRepository
// ---------------------------------------------------------------------------

func TestNewQdrantRepository_ValidAddress(t *testing.T) {
	embedder := &mockEmbedder{}
	// grpc.NewClient is lazy — it does not actually connect at construction time,
	// so a syntactically valid address should not return an error.
	repo, err := NewQdrantRepository("localhost:6334", embedder, "my_collection")
	if err != nil {
		t.Fatalf("expected no error for valid address, got: %v", err)
	}
	if repo == nil {
		t.Fatal("expected non-nil repository")
	}
	if repo.Collection != "my_collection" {
		t.Errorf("expected collection %q, got %q", "my_collection", repo.Collection)
	}
	if repo.Embedder != embedder {
		t.Error("expected embedder to be stored in repository")
	}
	if repo.Client == nil {
		t.Error("expected non-nil Qdrant client")
	}
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}