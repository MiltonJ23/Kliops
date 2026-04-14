package services 

import (
	"context"
	"fmt"
	"strings"

	"github.com/MiltonJ23/Kliops/internal/core/ports"
)

type KnowledgeService struct {
	VectorDB ports.KnowledgeBase
	DB       ports.IngestionRepository
} 

func NewKnowledgeService(vdb ports.KnowledgeBase, db ports.IngestionRepository) *KnowledgeService {
	return &KnowledgeService{
		VectorDB: vdb,
		DB:       db,
	}
}

// RetrieveRelevantMethodologies fetch pertinent historique reponses from Qdrant 
func (s *KnowledgeService) RetrieveRelevantMethodologies(ctx context.Context, exigence string) (string, error) {
	results, err := s.VectorDB.SearchSimilar(ctx, exigence, 3)
	if err != nil {
		return "", fmt.Errorf("erreur lors de la recherche vectorielle: %w", err)
	}

	if len(results) == 0 {
		return "Aucune information pertinente trouvée dans la mémoire technique de l'entreprise.", nil
	}

	var builder strings.Builder
	builder.WriteString("Voici le contexte technique extrait des anciens projets gagnés :\n\n")
	
	for i, res := range results {
		builder.WriteString(fmt.Sprintf("--- Référence %d (Score de pertinence: %.2f) ---\n", i+1, res.SimilarityScore))
		builder.WriteString(fmt.Sprintf("Exigence du CCTP passé : %s\n", res.ExigenceTechnique))
		builder.WriteString(fmt.Sprintf("Notre réponse validée : %s\n\n", res.ReponseApportee))
	}

	return builder.String(), nil
}