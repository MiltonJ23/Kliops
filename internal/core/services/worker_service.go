package services 

import (
	"context"
	"fmt"
	"log"
	"github.com/MiltonJ23/Kliops/internal/core/domain"
	"github.com/MiltonJ23/Kliops/internal/core/ports"
	"github.com/google/uuid"
)

type WorkerService struct {
	Repo ports.IngestionRepository
	LLM ports.LLMExtractor
	Knowledge ports.KnowledgeBase 
	Parser ports.DocumentParser
}

func NewWorkerService(repo ports.IngestionRepository,llm ports.LLMExtractor,kb ports.KnowledgeBase,parser ports.DocumentParser) *WorkerService {
	return &WorkerService{
		Repo:repo,
		LLM:llm,
		Knowledge:kb,
		Parser:parser,
	}
}

func (w *WorkerService) HandleJob(ctx context.Context,jobID,projectID string) error {

		// transit status to 'PROCESSING'
		updateJobStatusError := w.Repo.UpdateJobStatus(ctx,jobID,ports.JobProcessing,"")
		if updateJobStatusError != nil {
			return fmt.Errorf("failed to complete Job status: %v",updateJobStatusError)
		}

		dcePath, getDCEPathErr := w.Repo.GetDocumentPath(ctx,projectID,"DCE")
		if getDCEPathErr != nil {
			return w.failJob(ctx,jobID,fmt.Sprintf("failed to get DCE file path for project %s : %v",projectID,getDCEPathErr))
		}

		memPath, getMEMPathErr := w.Repo.GetDocumentPath(ctx,projectID,"MEMOIRE")
		if getMEMPathErr != nil {
			return w.failJob(ctx,jobID,fmt.Sprintf("failed to get MEM file path for project %s : %v",projectID,getMEMPathErr))
		}

		dceText,parsingDcePDFErr := w.Parser.FetchAndParse(ctx,dcePath)
		if parsingDcePDFErr != nil {
			return w.failJob(ctx,jobID,fmt.Sprintf("failed to parse DCE document `%s` in project `%s` : %v ",dcePath,projectID,parsingDcePDFErr))
		}

		memText,parsingMemPDFErr := w.Parser.FetchAndParse(ctx,memPath)
		if parsingMemPDFErr != nil {
			return w.failJob(ctx,jobID,fmt.Sprintf("failed to parse MEM document `%s` in project `%s` : %v ",dcePath,projectID,parsingMemPDFErr))
		}

		// let's extract it using Gemma damn 
		pairs, extractPairsFromGemmaErr := w.LLM.ExtractRequirementsAndAnswers(ctx,dceText,memText) 
		if extractPairsFromGemmaErr != nil {
			return w.failJob(ctx,jobID,fmt.Sprintf("failed to extract (exigence/reponse) pairs for project `%s` : %v",projectID,extractPairsFromGemmaErr))
		}

		// now we fed those pairs to qdrant 
		for _, pair := range pairs {
			qdrantID := uuid.New().String()

			reponseHistorique := domain.ReponseHistorique{
				ID: qdrantID,
				AppelOffreID: projectID,
				ExigenceTechnique: pair.Exigence,
				ReponseApportee: pair.Reponse,
				Gagne:	true ,
			}

			ingestionErr := w.Knowledge.Ingest(ctx,reponseHistorique)
			if ingestionErr != nil {
				return w.failJob(ctx,jobID,fmt.Sprintf("Vector DB ingestion failed : %v",ingestionErr))
			}

			savingPointInDatabaseErr := w.Repo.SaveResponseHistory(ctx,projectID,pair.Exigence,pair.Reponse,qdrantID)
			if savingPointInDatabaseErr != nil {
				return w.failJob(ctx,jobID,fmt.Sprintf("failed to save metadata of the project `%s` in the database: %v",projectID,savingPointInDatabaseErr))
			}
		}

		log.Printf("Job %s completed Successfully. %d pairs indexed",jobID,len(pairs))
		return w.Repo.UpdateJobStatus(ctx,jobID,ports.JobCompleted,"")

}



func (w *WorkerService) failJob(ctx context.Context,jobID,errMsg string) error {
	log.Printf("Job %s failed : %v",jobID,errMsg)
	w.Repo.UpdateJobStatus(ctx,jobID,ports.JobFailed,errMsg)
	return fmt.Errorf("%s", errMsg)
}