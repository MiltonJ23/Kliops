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
	embedding []float32
	err       error
	callCount int
	lastText  string
}

func (m *mockEmbedder) CreateEmbedding(_ context.Context, text string) ([]float32, error) {
	m.callCount++
	m.lastText = text
	return m.embedding, m.err
}

// ---------------------------------------------------------------------------
// Mock: qdrant.PointsClient
// The interface has many gRPC-generated methods; only Upsert and Search are
// exercised by QdrantRepository. All others return nil, nil.
// ---------------------------------------------------------------------------

type mockPointsClient struct {
	upsertErr    error
	searchResult *qdrant.SearchResponse
	searchErr    error

	upsertCalledWith  *qdrant.UpsertPoints
	searchCalledWith  *qdrant.SearchPoints
}

func (m *mockPointsClient) Upsert(_ context.Context, in *qdrant.UpsertPoints, _ ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	m.upsertCalledWith = in
	if m.upsertErr != nil {
		return nil, m.upsertErr
	}
	return &qdrant.PointsOperationResponse{}, nil
}

func (m *mockPointsClient) Search(_ context.Context, in *qdrant.SearchPoints, _ ...grpc.CallOption) (*qdrant.SearchResponse, error) {
	m.searchCalledWith = in
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	if m.searchResult != nil {
		return m.searchResult, nil
	}
	return &qdrant.SearchResponse{}, nil
}

// --- Unused methods to satisfy the qdrant.PointsClient interface ---

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

// Compile-time check that mockPointsClient satisfies the interface.
var _ qdrant.PointsClient = (*mockPointsClient)(nil)

// ---------------------------------------------------------------------------
// Helper: build a QdrantRepository with injected mocks
// ---------------------------------------------------------------------------

func newTestRepo(client qdrant.PointsClient, embedder ports.Embedder, collection string) *QdrantRepository {
	return &QdrantRepository{
		Client:     client,
		Embedder:   embedder,
		Collection: collection,
	}
}

// ---------------------------------------------------------------------------
// Ingest tests
// ---------------------------------------------------------------------------

func TestIngestSuccess(t *testing.T) {
	emb := &mockEmbedder{embedding: []float32{0.1, 0.2, 0.3}}
	client := &mockPointsClient{}
	repo := newTestRepo(client, emb, "test-collection")

	reponse := domain.ReponseHistorique{
		ID:                "uuid-001",
		AppelOffreID:      "AO-2024-001",
		ExigenceTechnique: "Pose de carrelage antidérapant",
		ReponseApportee:   "Utilisation de carrelage R11",
		Gagne:             true,
	}

	err := repo.Ingest(context.Background(), reponse)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Embedder was called with the technical requirement text
	if emb.callCount != 1 {
		t.Errorf("expected 1 embedding call, got %d", emb.callCount)
	}
	if emb.lastText != reponse.ExigenceTechnique {
		t.Errorf("expected embedding text %q, got %q", reponse.ExigenceTechnique, emb.lastText)
	}

	// Upsert was called with the correct collection
	if client.upsertCalledWith == nil {
		t.Fatal("expected Upsert to have been called")
	}
	if client.upsertCalledWith.CollectionName != "test-collection" {
		t.Errorf("expected collection test-collection, got %q", client.upsertCalledWith.CollectionName)
	}

	// Exactly one point was upserted
	if len(client.upsertCalledWith.Points) != 1 {
		t.Fatalf("expected 1 point, got %d", len(client.upsertCalledWith.Points))
	}
	point := client.upsertCalledWith.Points[0]

	// Point ID matches the reponse ID
	uuid, ok := point.Id.PointIdOptions.(*qdrant.PointId_Uuid)
	if !ok {
		t.Fatal("expected UUID point ID type")
	}
	if uuid.Uuid != "uuid-001" {
		t.Errorf("expected point UUID uuid-001, got %q", uuid.Uuid)
	}

	// Payload contains ao_id, reponse_apportee, gagne
	payload := point.Payload
	if payload["ao_id"].GetStringValue() != "AO-2024-001" {
		t.Errorf("payload ao_id mismatch: got %q", payload["ao_id"].GetStringValue())
	}
	if payload["reponse_apportee"].GetStringValue() != "Utilisation de carrelage R11" {
		t.Errorf("payload reponse_apportee mismatch: got %q", payload["reponse_apportee"].GetStringValue())
	}
	if !payload["gagne"].GetBoolValue() {
		t.Error("expected payload gagne=true")
	}
}

func TestIngestEmbedderError(t *testing.T) {
	emb := &mockEmbedder{err: errors.New("model unavailable")}
	client := &mockPointsClient{}
	repo := newTestRepo(client, emb, "col")

	err := repo.Ingest(context.Background(), domain.ReponseHistorique{ID: "x"})
	if err == nil {
		t.Fatal("expected error from embedder, got nil")
	}
	if client.upsertCalledWith != nil {
		t.Error("Upsert should not have been called when embedding fails")
	}
}

func TestIngestUpsertError(t *testing.T) {
	emb := &mockEmbedder{embedding: []float32{0.5}}
	client := &mockPointsClient{upsertErr: errors.New("qdrant write failed")}
	repo := newTestRepo(client, emb, "col")

	err := repo.Ingest(context.Background(), domain.ReponseHistorique{ID: "y"})
	if err == nil {
		t.Fatal("expected Upsert error to be propagated, got nil")
	}
	if !containsStr(err.Error(), "qdrant write failed") {
		t.Errorf("expected original error in message, got: %q", err.Error())
	}
}

func TestIngestVectorPassedToUpsert(t *testing.T) {
	wantVec := []float32{0.11, 0.22, 0.33}
	emb := &mockEmbedder{embedding: wantVec}
	client := &mockPointsClient{}
	repo := newTestRepo(client, emb, "col")

	_ = repo.Ingest(context.Background(), domain.ReponseHistorique{ID: "z"})

	if client.upsertCalledWith == nil {
		t.Fatal("Upsert not called")
	}
	point := client.upsertCalledWith.Points[0]
	vec := point.Vectors.GetVector().Data
	if len(vec) != len(wantVec) {
		t.Fatalf("expected vector length %d, got %d", len(wantVec), len(vec))
	}
	for i, v := range vec {
		if v != wantVec[i] {
			t.Errorf("vector[%d]: expected %f, got %f", i, wantVec[i], v)
		}
	}
}

func TestIngestGagneFalse(t *testing.T) {
	emb := &mockEmbedder{embedding: []float32{0.1}}
	client := &mockPointsClient{}
	repo := newTestRepo(client, emb, "col")

	err := repo.Ingest(context.Background(), domain.ReponseHistorique{
		ID:    "rh-false",
		Gagne: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	payload := client.upsertCalledWith.Points[0].Payload
	if payload["gagne"].GetBoolValue() {
		t.Error("expected gagne=false in payload")
	}
}

// ---------------------------------------------------------------------------
// SearchSimilar tests
// ---------------------------------------------------------------------------

func buildSearchResponse(items []struct {
	reponse string
	gagne   bool
	score   float32
}) *qdrant.SearchResponse {
	hits := make([]*qdrant.ScoredPoint, len(items))
	for i, item := range items {
		hits[i] = &qdrant.ScoredPoint{
			Score: item.score,
			Payload: map[string]*qdrant.Value{
				"reponse_apportee": {Kind: &qdrant.Value_StringValue{StringValue: item.reponse}},
				"gagne":            {Kind: &qdrant.Value_BoolValue{BoolValue: item.gagne}},
			},
		}
	}
	return &qdrant.SearchResponse{Result: hits}
}

func TestSearchSimilarSuccess(t *testing.T) {
	emb := &mockEmbedder{embedding: []float32{0.9, 0.1}}
	client := &mockPointsClient{
		searchResult: buildSearchResponse([]struct {
			reponse string
			gagne   bool
			score   float32
		}{
			{"Réponse A", true, 0.92},
			{"Réponse B", false, 0.75},
		}),
	}
	repo := newTestRepo(client, emb, "knowledge")

	results, err := repo.SearchSimilar(context.Background(), "nouvelle exigence", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].ReponseApportee != "Réponse A" {
		t.Errorf("first result ReponseApportee: got %q", results[0].ReponseApportee)
	}
	if results[0].SimilarityScore != 0.92 {
		t.Errorf("first result score: expected 0.92, got %f", results[0].SimilarityScore)
	}
	if !results[0].Gagne {
		t.Error("first result Gagne should be true")
	}

	if results[1].ReponseApportee != "Réponse B" {
		t.Errorf("second result ReponseApportee: got %q", results[1].ReponseApportee)
	}
	if results[1].SimilarityScore != 0.75 {
		t.Errorf("second result score: expected 0.75, got %f", results[1].SimilarityScore)
	}
	if results[1].Gagne {
		t.Error("second result Gagne should be false")
	}
}

func TestSearchSimilarEmbedderError(t *testing.T) {
	emb := &mockEmbedder{err: errors.New("embedding model offline")}
	client := &mockPointsClient{}
	repo := newTestRepo(client, emb, "col")

	results, err := repo.SearchSimilar(context.Background(), "query", 5)
	if err == nil {
		t.Fatal("expected error from embedder, got nil")
	}
	if results != nil {
		t.Errorf("expected nil results on error, got %v", results)
	}
	if client.searchCalledWith != nil {
		t.Error("Search should not have been called when embedding fails")
	}
}

func TestSearchSimilarQdrantError(t *testing.T) {
	emb := &mockEmbedder{embedding: []float32{0.5}}
	client := &mockPointsClient{searchErr: errors.New("qdrant read failed")}
	repo := newTestRepo(client, emb, "col")

	results, err := repo.SearchSimilar(context.Background(), "query", 3)
	if err == nil {
		t.Fatal("expected search error to propagate, got nil")
	}
	if results != nil {
		t.Errorf("expected nil results on error, got %v", results)
	}
}

func TestSearchSimilarSendsCorrectParameters(t *testing.T) {
	wantVec := []float32{0.7, 0.3}
	emb := &mockEmbedder{embedding: wantVec}
	client := &mockPointsClient{}
	repo := newTestRepo(client, emb, "my-collection")

	_, _ = repo.SearchSimilar(context.Background(), "Poser du carrelage", 7)

	if client.searchCalledWith == nil {
		t.Fatal("Search was not called")
	}
	req := client.searchCalledWith
	if req.CollectionName != "my-collection" {
		t.Errorf("expected collection my-collection, got %q", req.CollectionName)
	}
	if req.Limit != 7 {
		t.Errorf("expected limit 7, got %d", req.Limit)
	}
	if len(req.Vector) != len(wantVec) {
		t.Fatalf("vector length mismatch: expected %d, got %d", len(wantVec), len(req.Vector))
	}
	for i, v := range req.Vector {
		if v != wantVec[i] {
			t.Errorf("vector[%d]: expected %f, got %f", i, wantVec[i], v)
		}
	}
}

func TestSearchSimilarEmptyResults(t *testing.T) {
	emb := &mockEmbedder{embedding: []float32{0.1}}
	client := &mockPointsClient{
		searchResult: &qdrant.SearchResponse{Result: []*qdrant.ScoredPoint{}},
	}
	repo := newTestRepo(client, emb, "col")

	results, err := repo.SearchSimilar(context.Background(), "no match query", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Result should be nil or empty slice (no hits)
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchSimilarPayloadEnabled(t *testing.T) {
	emb := &mockEmbedder{embedding: []float32{0.1}}
	client := &mockPointsClient{}
	repo := newTestRepo(client, emb, "col")

	_, _ = repo.SearchSimilar(context.Background(), "query", 1)

	req := client.searchCalledWith
	if req == nil {
		t.Fatal("Search was not called")
	}
	if req.WithPayload == nil {
		t.Fatal("WithPayload selector not set")
	}
	enable, ok := req.WithPayload.SelectorOptions.(*qdrant.WithPayloadSelector_Enable)
	if !ok {
		t.Fatal("expected WithPayloadSelector_Enable type")
	}
	if !enable.Enable {
		t.Error("expected payload to be enabled in search request")
	}
}

func TestSearchSimilarEmbedderCalledWithQuery(t *testing.T) {
	emb := &mockEmbedder{embedding: []float32{0.5}}
	client := &mockPointsClient{}
	repo := newTestRepo(client, emb, "col")

	_, _ = repo.SearchSimilar(context.Background(), "recherche spécifique", 1)

	if emb.lastText != "recherche spécifique" {
		t.Errorf("expected embedder to be called with query text, got %q", emb.lastText)
	}
}

// ---------------------------------------------------------------------------
// NewQdrantRepository constructor test
// ---------------------------------------------------------------------------

func TestNewQdrantRepositoryStoresFields(t *testing.T) {
	// grpc.NewClient with a valid address should not fail (lazy connection).
	emb := &mockEmbedder{}
	repo, err := NewQdrantRepository("localhost:6334", emb, "my-col")
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	if repo.Collection != "my-col" {
		t.Errorf("expected collection my-col, got %q", repo.Collection)
	}
	if repo.Embedder != emb {
		t.Error("expected embedder to be stored")
	}
	if repo.Client == nil {
		t.Error("expected non-nil qdrant client")
	}
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func containsStr(s, substr string) bool {
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