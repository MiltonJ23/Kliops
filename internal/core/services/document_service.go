package services

import (
	"context"
	"fmt"

	"github.com/MiltonJ23/Kliops/internal/core/ports"
)

const (
	configBucket      = "kliops-config"
	templateObjectKey = "template_charte.docx"
)

type DocumentService struct {
	Storage   ports.FileStorage
	Generator ports.DocumentGenerator
}

func NewDocumentService(storage ports.FileStorage, docGen ports.DocumentGenerator) *DocumentService {
	return &DocumentService{
		Storage:   storage,
		Generator: docGen,
	}
}

func (d *DocumentService) CompileTechnicalMemory(ctx context.Context, projectName string, variables map[string]string, targetEmail string) (string, error) {
	templateStream, streamingErr := d.Storage.DownloadStream(ctx, configBucket, templateObjectKey)
	if streamingErr != nil {
		return "", fmt.Errorf("failed to stream the template %s/%s from storage: %w", configBucket, templateObjectKey, streamingErr)
	}

	defer templateStream.Close()

	docName := fmt.Sprintf("Mémoire Technique - %s", projectName)

	docID, docURL, err := d.Generator.GenerateFromStream(ctx, templateStream, docName, variables)
	if err != nil {
		return "", fmt.Errorf("document generation failed: %w", err)
	}

	err = d.Generator.ShareWithUser(ctx, docID, targetEmail)
	if err != nil {
		return docURL, fmt.Errorf("document generated but sharing failed: %w", err)
	}

	return docURL, nil
}
