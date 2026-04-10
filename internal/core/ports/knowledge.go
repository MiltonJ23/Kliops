package ports

import (
	"context"

	"github.com/MiltonJ23/Kliops/internal/core/domain"
)

// SearchResult represents what the knowledge base returns to the agent.
type SearchResult struct {
	domain.ReponseHistorique
	SimilarityScore float32 // ranging from 0.0 to 1.0 (1.0 being the perfect match)
}

// KnowledgeBase is the contract the vector base is supposed to honour
type KnowledgeBase interface {

	// Ingest saves a new ReponseHistorique in the knowledge base
	Ingest(ctx context.Context, reponse domain.ReponseHistorique) error

	// SearchSimilar searches for past responses based on a new CCTP requirement
	SearchSimilar(ctx context.Context, nouvelleExigence string, limit int) ([]SearchResult, error)
}

// Embedder is the contract to transform text into vectors
type Embedder interface {
	CreateEmbedding(ctx context.Context, text string) ([]float32, error)
}
