package ports

import (
	"context"
)

type JobStatus string

const (
	JobPending    JobStatus = "PENDING"
	JobProcessing JobStatus = "PROCESSING"
	JobCompleted  JobStatus = "COMPLETED"
	JobFailed     JobStatus = "FAILED"
)

type ProjectManifest struct {
	ExternalID string
	Titre      string
	Client     string
	Status     string
	FichierDCE string
	FichierMEM string
}

type ExtractedPair struct {
	Exigence string `json:"exigence"`
	Reponse  string `json:"reponse"`
}

type IngestionRepository interface {
	// ExecuteTx is the transactionality warrant
	ExecuteTx(ctx context.Context, fn func(txRepo IngestionRepository) error) error
	CheckProjectExists(ctx context.Context, externalID string) (bool, error)
	CreateProject(ctx context.Context, p ProjectManifest) (string, error)
	SaveDocument(ctx context.Context, projectId, docType, minioPath string) error
	CreateJob(ctx context.Context, projectID string) error
	UpdateJobStatus(ctx context.Context, jobId string, status JobStatus, errMsg string) error
	SaveReponseHistorique(ctx context.Context, projectID, exigence, reponse, qdrantID string) error
}

type MessageQueue interface {
	PublishJob(ctx context.Context, jobID, projectID string) error
	ConsumeJob(ctx context.Context, handler func(ctx context.Context, jobID, projectID string) error) error
}

type LLMExtractor interface {
	ExtractRequirementsAndAnswers(ctx context.Context, dceText, memoireText string) ([]ExtractedPair, error)
}
