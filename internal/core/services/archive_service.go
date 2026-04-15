package services

import (
	"archive/zip"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/MiltonJ23/Kliops/internal/core/ports"
)

type ArchiveService struct {
	Repo    ports.IngestionRepository
	Storage ports.FileStorage
	Queue   ports.MessageQueue
}

func NewArchiveService(repo ports.IngestionRepository, storage ports.FileStorage, queue ports.MessageQueue) *ArchiveService {
	return &ArchiveService{Repo: repo, Storage: storage, Queue: queue}
}

func (s *ArchiveService) UploadFileFromZip(ctx context.Context, z *zip.ReadCloser, filename, projectId, docType string) ([]string, error) {
	for _, f := range z.File {
		if filepath.Base(f.Name) == filename {
			rc, openingFileError := f.Open()
			if openingFileError != nil {
				return nil, fmt.Errorf("an error occurred while opening the file %s to be uploaded into the MiniO: %v", filename, openingFileError)
			}
			defer rc.Close()

			contentType := mime.TypeByExtension(filepath.Ext(filename))
			if contentType == "" {
				contentType = "application/octet-stream"
			}

			path, uploadingFileError := s.Storage.Upload(ctx, "dce-archive", fmt.Sprintf("%s/%s/%s", projectId, strings.ToLower(docType), filename), rc, int64(f.UncompressedSize64), contentType)
			if uploadingFileError != nil {
				return nil, fmt.Errorf("an error occurred while uploading file %s , into the bucket:%v", filename, uploadingFileError)
			}
			return []string{path}, nil
		}
	}
	return nil, fmt.Errorf("file %s referenced in manifest is not found in zip", filename)
}

func (s *ArchiveService) ProcessZipArchive(ctx context.Context, file multipart.File, size int64) error {
	const maxZipSize = 500 * 1024 * 1024 // 500MB
	if size > maxZipSize {
		return fmt.Errorf("zip file too large: %d bytes", size)
	}

	tempFile, err := os.CreateTemp("", "archive-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		return fmt.Errorf("failed to copy to temp file: %v", err)
	}

	zipReader, err := zip.OpenReader(tempFile.Name())
	if err != nil {
		return fmt.Errorf("failed to open zip reader: %v", err)
	}
	defer zipReader.Close()

	// we then extract the manifest.csv from the zip file

	var manifestFile *zip.File

	for _, f := range zipReader.File {
		if filepath.Base(f.Name) == "manifest.csv" {
			manifestFile = f
			break
		}
	}

	if manifestFile == nil {
		return fmt.Errorf("manifest.csv not found in the archive ZIP")
	}

	rc, openingManifestCSVFileError := manifestFile.Open()
	if openingManifestCSVFileError != nil {
		return fmt.Errorf("an error occurred while opening the manifest file: %v", openingManifestCSVFileError)
	}
	defer rc.Close()

	csvReader := csv.NewReader(rc)
	records, readAllCSVHeadError := csvReader.ReadAll() // all of them manifest head : id_projet, titre, client, statut, fichier_dce, fichier_memoire
	if readAllCSVHeadError != nil {
		return fmt.Errorf("unable to read the manifest.csv file , an error occurred : %v", readAllCSVHeadError)
	}

	for i, record := range records {
		if i == 0 {
			continue // we skip since it is the header
		}

		// Validate record has expected columns
		if len(record) < 6 {
			log.Printf("Skipping malformed CSV row: expected at least 6 columns, got %d", len(record))
			continue
		}

		manifest := ports.ProjectManifest{
			ExternalID: record[0], Titre: record[1], Client: record[2], Status: record[3], FichierDCE: record[4], FichierMEM: record[5],
		}
		// let's check if the project doesn't exist already
		exists, checkErr := s.Repo.CheckProjectExists(ctx, manifest.ExternalID)
		if checkErr != nil {
			log.Printf("Error checking project existence: %v", checkErr)
			continue
		}
		if exists {
			continue
		}

		// Now let's launch a new Transaction
		var jobID, projID string
		var uploadedPaths []string
		transactionError := s.Repo.ExecuteTx(ctx, func(txRepo ports.IngestionRepository) error {
			// we first create the project
			var projectCreationError error
			projID, projectCreationError = txRepo.CreateProject(ctx, manifest)
			if projectCreationError != nil {
				return fmt.Errorf("an error occurred while trying to create a new project from the manifest")
			}
			//we then extract the pdf from the zip and send them to MiniO

			paths, uploadingZipFileError := s.UploadFileFromZip(ctx, zipReader, manifest.FichierDCE, projID, "DCE")
			if uploadingZipFileError != nil {
				return fmt.Errorf("an error occurred while trying to upload all of the zip files to the bucket")
			}
			uploadedPaths = append(uploadedPaths, paths...)
			for _, path := range paths {
				err := txRepo.SaveDocument(ctx, projID, "DCE", path)
				if err != nil {
					return fmt.Errorf("an error occurred while saving document")
				}
			}

			// we then create the Job
			var jobCreationError error
			jobID, jobCreationError = txRepo.CreateJob(ctx, projID)
			if jobCreationError != nil {
				return fmt.Errorf("an error occurred while trying to create a new job")
			}
			return nil
		})
		if transactionError != nil {
			log.Printf("Failed to process project %s: %v", manifest.ExternalID, transactionError)
			for _, path := range uploadedPaths {
				parts := strings.Split(strings.TrimPrefix(path, "minio://"), "/")
				if len(parts) >= 2 {
					bucket := parts[0]
					object := strings.Join(parts[1:], "/")
					s.Storage.Delete(ctx, bucket, object)
				}
			}
			continue
		}

		// Publish the job after the transaction succeeds
		if publishErr := s.Queue.PublishJob(ctx, jobID, projID); publishErr != nil {
			log.Printf("Error publishing job %s for project %s: %v", jobID, projID, publishErr)
			// Log but don't fail the whole process as the job is already in the DB
		}
	}
	return nil
}
