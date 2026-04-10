package repositories 

import (
	"context"
	"errors"
	"fmt"
	"github.com/MiltonJ23/Kliops/internal/core/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)


// DBTX abstract a postgres transaction, to allow to execute request whether in a Tx or Not 
type DBTX interface {
	Exec(ctx context.Context,sql string,arguments ...any)(pgconn.CommandTag,error)
	QueryRow(ctx context.Context,sql string, arguments ...any) pgx.Row
	Begin(ctx context.Context) ( pgx.Tx,error)
}


type IngestionPostgres struct {
	DB DBTX 
}

func NewIngestionPostgres(db *pgxpool.Pool) *IngestionPostgres {
	return &IngestionPostgres{DB:db}
}


var _ ports.IngestionRepository = (*IngestionPostgres)(nil)


func (r *IngestionPostgres) ExecuteTx(ctx context.Context, fn func(txRepo ports.IngestionRepository) error) error {
	tx, transactionBeginError := r.DB.Begin(ctx)
	if transactionBeginError != nil {
		return fmt.Errorf("failed to begin a new transaction: %w",transactionBeginError)
	}

	// we create a new instance of the repository that is going to use the transaction rather than the pool 
	txRepo := &IngestionPostgres{DB:tx}

	trsError := fn(txRepo) 
	if trsError != nil {
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
				return fmt.Errorf("tx error: %w , rollback error : %v",trsError,rollbackErr)
		}
		return trsError
	}
	return tx.Commit(ctx)
}

func (r *IngestionPostgres) CheckProjectExists(ctx context.Context, externalID string) (bool, error) {
	var exists bool 
	query := `SELECT EXISTS(SELECT 1 from appels_offres WHERE external_id=$1)`
	executingQueryError := r.DB.QueryRow(ctx,query,externalID).Scan(&exists)

	return exists, executingQueryError
}

func (r *IngestionPostgres) CreateProject(ctx context.Context, p ports.ProjectManifest) (string, error) {
	var id string 
	query := `INSERT INTO appels_offres(external_id,titre,client,status)
				VALUES ($1,$2,$3,$4) 
				RETURNING id
	`
	executingQueryError := r.DB.QueryRow(ctx,query,p.ExternalID,p.Titre,p.Client,p.Status).Scan(&id)
	if executingQueryError != nil {
		return "", fmt.Errorf("failed to create project %s , an error occured: %v",p.ExternalID,executingQueryError)
	}
	
	return id,nil
}

func (r *IngestionPostgres) SaveDocument(ctx context.Context, projectId, docType, minioPath string) error {
	query := `INSERT INTO documents (appel_offre_id,type,minio_path)
				VALUES ($1,$2,$3)
	`
	_, err := r.DB.Exec(ctx,query,projectId,docType,minioPath)
	return err
}

func (r *IngestionPostgres) CreateJob(ctx context.Context, projectID string) (string,error) {
	var jobID string
	query := `
		INSERT INTO processing_jobs(appel_offre_id,status)
		VALUES($1,'PENDING')
		RETURNING id
	` 
	executionQueryError := r.DB.QueryRow(ctx,query,projectID).Scan(&jobID)
	return jobID,executionQueryError
}

func (r *IngestionPostgres) UpdateJobStatus(ctx context.Context, jobId string, status ports.JobStatus, errMsg string) error {
	query := `
		UPDATE processing_jobs 
		SET status = $1, error_message = $2, updated_at=NOW()
		WHERE id=$3
	`
	_, updateJobStatusError := r.DB.Exec(ctx,query,status,errMsg,jobId)
	return updateJobStatusError
}

func (r *IngestionPostgres) SaveResponseHistory(ctx context.Context, projectID, exigence, response, qdrantID string) error {
	query := `
		INSERT INTO reponses_historiques (appel_offre_id,exigence_technique,reponse_apportee,qdrant_point_id)
		VALUES ($1,$2,$3,$4)
	`
	_,savingResponseHistoryError := r.DB.Exec(ctx,query,projectID,exigence,response,qdrantID)
	return savingResponseHistoryError
}

func (r *IngestionPostgres) GetDocumentPath(ctx context.Context,projectID,docType string) (string,error) {
	var path string 
	query := `SELECT minio_path FROM documents WHERE appel_offre_id=$1 AND type = $2 LIMIT 1`

	executingFetchDocumentError := r.DB.QueryRow(ctx,query,projectID,docType).Scan(&path)

	if executingFetchDocumentError != nil {
			if errors.Is(executingFetchDocumentError,pgx.ErrNoRows){
					return "", fmt.Errorf("document not found for project %s with type %s", projectID, docType)
			}
			return "",executingFetchDocumentError
	}
	return path,nil
}
