package repositories

import (
	"fmt"
	"context"
	"github.com/MiltonJ23/Kliops/internal/core/domain"
	"github.com/MiltonJ23/Kliops/internal/core/ports"
	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type QdrantRepository struct {
	Client qdrant.PointsClient 
	Embedder ports.Embedder 
	Collection string 
}


// NewQdrantRepository creates a QdrantRepository configured to talk to the Qdrant instance at qdrantAddr.
// It opens a gRPC connection using insecure transport credentials, instantiates a Qdrant PointsClient,
// and returns the repository configured with the provided embedder and collection name or an error if the connection fails.
func NewQdrantRepository(qdrantAddr string, embedder ports.Embedder, collectionName string) (*QdrantRepository,error) {
	// first of all, we connect to qdrant through gRPC 
	conn, connectionError := grpc.NewClient(qdrantAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if connectionError != nil {
		return nil, fmt.Errorf("unable to connect to Qdrant %v",connectionError)
	}

	client := qdrant.NewPointsClient(conn)

	return &QdrantRepository{
		Client: client,
		Embedder: embedder,
		Collection: collectionName,
	},nil
}

// Ingest will vectorize the new requirement and save the response in Qdrant
func (r *QdrantRepository) Ingest(ctx context.Context, reponse domain.ReponseHistorique) error {
	// let's convert the new requirement into a math vector 
	vector, vectorizationError := r.Embedder.CreateEmbedding(ctx,reponse.ExigenceTechnique)
	if vectorizationError != nil {
		return fmt.Errorf("an error occured during the vectorization of the requirement: %v",vectorizationError)
	}

	payload := map[string]*qdrant.Value{
		"ao_id": {Kind:&qdrant.Value_StringValue{StringValue: reponse.AppelOffreID}},
		"reponse_apportee" : {Kind: &qdrant.Value_StringValue{StringValue: reponse.ReponseApportee}},
		"gagne" : {Kind: &qdrant.Value_BoolValue{BoolValue: reponse.Gagne}},
	}

	//now let's try to insert that into qdrant 
	points := []*qdrant.PointStruct{
		{
			Id: &qdrant.PointId{PointIdOptions: &qdrant.PointId_Uuid{Uuid: reponse.ID}},
			Vectors: &qdrant.Vectors{VectorsOptions: &qdrant.Vectors_Vector{Vector: &qdrant.Vector{Data: vector}}},
			Payload: payload,

		},
	}

	_, upsertingVectorsError := r.Client.Upsert(ctx, &qdrant.UpsertPoints{
				CollectionName: r.Collection,
				Points: points,
	})

	if upsertingVectorsError != nil {
		return fmt.Errorf("an error occured while inserting into Qdrant: %v",upsertingVectorsError)
	}
	return nil
}


func (r *QdrantRepository) SearchSimilar(ctx context.Context, nouvelleExigence string, limit int) ([]ports.SearchResult,error) {
	// first of all , let's vectorize the request 
	queryVector, queryVectorizationError := r.Embedder.CreateEmbedding(ctx,nouvelleExigence)
	if queryVectorizationError != nil {
		return nil, fmt.Errorf("an error occured while vectorizing the request: %v",queryVectorizationError)
	}

	// next thing we do is to search in  qdrant 
	searchResult, searchingVectorsError := r.Client.Search(ctx, &qdrant.SearchPoints{
		CollectionName: r.Collection,
		Vector: queryVector,
		Limit: uint64(limit),
		WithPayload: &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
	})

	if searchingVectorsError != nil {
		return nil, fmt.Errorf("an error occured while searching for Vector Result in Qdrant: %v",searchingVectorsError)
	}

	// now let's map the result we got from Qdrant 
	var results []ports.SearchResult 
	for _,hit := range searchResult.Result{
		payload := hit.Payload 

		results = append(results,ports.SearchResult{
			ReponseHistorique: domain.ReponseHistorique{
				ReponseApportee: payload["reponse_apportee"].GetStringValue(),
				Gagne: payload["gagne"].GetBoolValue(),
			},
			SimilarityScore: hit.Score,
		})
	}

	return results,nil
}