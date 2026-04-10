package repositories

import (
	"context"
	"fmt"

	"github.com/MiltonJ23/Kliops/internal/core/domain"
	"github.com/MiltonJ23/Kliops/internal/core/ports"
	"github.com/qdrant/go-client/qdrant"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type QdrantRepository struct {
	Client     qdrant.PointsClient
	Conn       *grpc.ClientConn
	Embedder   ports.Embedder
	Collection string
}

func NewQdrantRepository(qdrantAddr string, embedder ports.Embedder, collectionName string) (*QdrantRepository, error) {
	if embedder == nil {
		return nil, fmt.Errorf("embedder is required")
	}
	// first of all, we connect to qdrant through gRPC
	conn, connectionError := grpc.NewClient(qdrantAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if connectionError != nil {
		return nil, fmt.Errorf("unable to connect to Qdrant %v", connectionError)
	}

	client := qdrant.NewPointsClient(conn)

	return &QdrantRepository{
		Client:     client,
		Conn:       conn,
		Embedder:   embedder,
		Collection: collectionName,
	}, nil
}

func (r *QdrantRepository) Close() error {
	if r.Conn != nil {
		return r.Conn.Close()
	}
	return nil
}

// Ingest will vectorize the new requirement and save the response in Qdrant
func (r *QdrantRepository) Ingest(ctx context.Context, reponse domain.ReponseHistorique) error {
	// let's convert the new requirement into a math vector
	vector, vectorizationError := r.Embedder.CreateEmbedding(ctx, reponse.ExigenceTechnique)
	if vectorizationError != nil {
		return fmt.Errorf("an error occured during the vectorization of the requirement: %v", vectorizationError)
	}

	payload := map[string]*qdrant.Value{
		"ao_id":              {Kind: &qdrant.Value_StringValue{StringValue: reponse.AppelOffreID}},
		"reponse_apportee":   {Kind: &qdrant.Value_StringValue{StringValue: reponse.ReponseApportee}},
		"gagne":              {Kind: &qdrant.Value_BoolValue{BoolValue: reponse.Gagne}},
		"exigence_technique": {Kind: &qdrant.Value_StringValue{StringValue: reponse.ExigenceTechnique}},
	}

	//now let's try to insert that into qdrant
	points := []*qdrant.PointStruct{
		{
			Id:      &qdrant.PointId{PointIdOptions: &qdrant.PointId_Uuid{Uuid: reponse.ID}},
			Vectors: &qdrant.Vectors{VectorsOptions: &qdrant.Vectors_Vector{Vector: &qdrant.Vector{Data: vector}}},
			Payload: payload,
		},
	}

	_, upsertingVectorsError := r.Client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: r.Collection,
		Points:         points,
	})

	if upsertingVectorsError != nil {
		return fmt.Errorf("an error occured while inserting into Qdrant: %v", upsertingVectorsError)
	}
	return nil
}

func (r *QdrantRepository) SearchSimilar(ctx context.Context, nouvelleExigence string, limit int) ([]ports.SearchResult, error) {
	// Validate limit parameter
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be positive, got %d", limit)
	}

	// first of all , let's vectorize the request
	queryVector, queryVectorizationError := r.Embedder.CreateEmbedding(ctx, nouvelleExigence)
	if queryVectorizationError != nil {
		return nil, fmt.Errorf("an error occured while vectorizing the request: %v", queryVectorizationError)
	}

	// next thing we do is to search in  qdrant
	searchResult, searchingVectorsError := r.Client.Search(ctx, &qdrant.SearchPoints{
		CollectionName: r.Collection,
		Vector:         queryVector,
		Limit:          uint64(limit),
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
	})

	if searchingVectorsError != nil {
		return nil, fmt.Errorf("an error occured while searching for Vector Result in Qdrant: %v", searchingVectorsError)
	}

	// now let's map the result we got from Qdrant
	var results []ports.SearchResult
	for _, hit := range searchResult.Result {
		payload := hit.Payload

		// Safely extract payload values with nil checks
		aoID := ""
		if payload != nil && payload["ao_id"] != nil {
			aoID = payload["ao_id"].GetStringValue()
		}

		reponseApportee := ""
		if payload != nil && payload["reponse_apportee"] != nil {
			reponseApportee = payload["reponse_apportee"].GetStringValue()
		}

		gagne := false
		if payload != nil && payload["gagne"] != nil {
			gagne = payload["gagne"].GetBoolValue()
		}

		id := ""
		if payload != nil && payload["id"] != nil {
			id = payload["id"].GetStringValue()
		}

		exigenceTechnique := ""
		if payload != nil && payload["exigence_technique"] != nil {
			exigenceTechnique = payload["exigence_technique"].GetStringValue()
		}

		results = append(results, ports.SearchResult{
			ReponseHistorique: domain.ReponseHistorique{
				ID:                id,
				AppelOffreID:      aoID,
				ExigenceTechnique: exigenceTechnique,
				ReponseApportee:   reponseApportee,
				PrixPropose:       decimal.Zero,
				Gagne:             gagne,
			},
			SimilarityScore: hit.Score,
		})
	}

	return results,nil
}

func (r *QdrantRepository) DeleteByIDs(ctx context.Context, ids []string) error {
	// TODO: implement delete by IDs
	return fmt.Errorf("DeleteByIDs not implemented")
}
