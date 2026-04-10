package main

import (
	"context"
	"fmt"
	"log"
	"github.com/MiltonJ23/Kliops/internal/adapters/llm"
	"github.com/MiltonJ23/Kliops/internal/adapters/repositories"
	"github.com/MiltonJ23/Kliops/internal/core/domain"
	"github.com/google/uuid"
)


func main() {
	ctx := context.Background()

	embedder := llm.NewOllamaEmbedder("http://localhost:11434","mxbai-embed-large")

	repo, err := repositories.NewQdrantRepository("localhost:6334",embedder,"memoire_technique")
	if err != nil {
		log.Fatalf("failed to initialize Qdrant : %v",err)
	}

	fmt.Println("--> Done initializing the components. Let's start the ingestion right away ....")

	archive := domain.ReponseHistorique{
		ID:                uuid.New().String(), 
		AppelOffreID:      "AO-2023-BTP-LYON",
		ExigenceTechnique: "L'entrepreneur devra détailler la méthodologie employée pour la pose de revêtements de sol souples en milieu hospitalier, en respectant les normes d'hygiène.",
		ReponseApportee:   "Nous utiliserons du PVC homogène soudé à chaud. Nos équipes déploieront un sas de confinement antipoussière de classe 3 et procéderont à une décontamination bi-journalière selon la norme NF S90-351.",
		Gagne:             true,
	}

	err = repo.Ingest(ctx, archive)
	if err != nil {
		log.Fatalf("Failed the ingestion: %v", err)
	}

	fmt.Println("=> Data inserted successfully into Qdrant !")

	fmt.Println("\n=> a new Call for Tenders is arriving. We question the MCP ....")

	nouvelleQuestionDuClient := "Quel type de sol allez-vous poser dans les couloirs de la clinique et comment gérez-vous la poussière ?"
	
	resultats, err := repo.SearchSimilar(ctx, nouvelleQuestionDuClient, 1)
	if err != nil {
		log.Fatalf("Échec de la recherche: %v", err)
	}
	
	if len(resultats) == 0 {
		log.Println("No similar responses found in knowledge base")
		return
	}

	fmt.Printf("\n--- RESULT OF THE RAG (Similarity Score: %f) ---\n", resultats[0].SimilarityScore)
	fmt.Printf("Réponse historique found :\n%s\n", resultats[0].ReponseApportee)
	fmt.Println("-----------------------------------------------------")

}

