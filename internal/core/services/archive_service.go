package services 

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"mime/multipart"
	"io"
	"path/filepath"
	"github.com/MiltonJ23/Kliops/internal/core/ports"
)

type ArchiveService struct {
	Repo ports.IngestionRepository
	Storage ports.FileStorage 
	Queue ports.MessageQueue
}


func NewArchiveService(repo ports.IngestionRepository,storage ports.FileStorage,queue ports.MessageQueue) *ArchiveService {
	return &ArchiveService{Repo:repo,Storage:storage,Queue:queue}
}

func (s *ArchiveService) UploadFileFromZip(ctx context.Context,z *zip.Reader,filename,projectId,docType string,txRepo ports.IngestionRepository) error {
	for _,f := range z.File{
		if filepath.Base(f.Name) == filename {
			rc , openingFileError := f.Open()
			if openingFileError != nil {
				return fmt.Errorf("an error occured while opening the file %s to be uploaded into the MiniO: %v",filename,openingFileError)
			}
			defer rc.Close()

			path, uploadingFileError := s.Storage.Upload(ctx,"dce-archive",fmt.Sprintf("%s/%s",projectId,filename),rc,int64(f.UncompressedSize64),"application/pdf")
			if uploadingFileError != nil {
				return fmt.Errorf("an error occured while uploading file %s , into the bucket:%v",filename,uploadingFileError)
			}
			return txRepo.SaveDocument(ctx,projectId,docType,path)
		}
	}
	return fmt.Errorf("file %s referenced in manifest is not found in zip",filename)
}


func (s *ArchiveService) ProcessZipArchive(ctx context.Context,file multipart.File,size int64) error {
	zipData,fileLoadingError := io.ReadAll(file)
	if fileLoadingError != nil {
		return fmt.Errorf("an occured while loading the Zip file : %v",fileLoadingError)
	}
	//TODO: make sure to use os.File rather than multipart for zip files of more than 500 MB to avoid clogging up the RAM

	zipReader,openingNewZipReaderError := zip.NewReader(bytes.NewReader(zipData),size)
	if openingNewZipReaderError != nil {
		return fmt.Errorf("an error occured while trying to open a new zip reader : %v",openingNewZipReaderError)
	}
	// we then extract the manifest.csv from the zip file 
	
	var manifestFile *zip.File 

	for _,f := range zipReader.File{
		if filepath.Base(f.Name) == "manifest.csv" {
			manifestFile = f
			break
		}
	}

	if manifestFile == nil {
		return fmt.Errorf("manifest.csv not found in the archive ZIP")
	}

	rc,openingManifestCSVFileError := manifestFile.Open()
	if openingManifestCSVFileError != nil {
		return fmt.Errorf("an error occured while opening the manifest file: %v",openingManifestCSVFileError)
	}
	defer rc.Close()

	csvReader := csv.NewReader(rc)
	records, readAllCSVHeadError := csvReader.ReadAll() // all of them manifest head : id_projet, titre, client, statut, fichier_dce, fichier_memoire
	if readAllCSVHeadError != nil {
		return fmt.Errorf("unable to read the manifest.csv file , an error occurred : %v",readAllCSVHeadError)
	}

	for i, record := range records {
		if i == 0 {
			continue // we skip since it is the header s
		}

		manifest := ports.ProjectManifest{
			ExternalID:record[0],Titre:record[1],Client:record[2],Status:record[3],FichierDCE:record[4],FichierMEM:record[5],
		}
		// let's check if the project doesn't exist already 
		exists, _ := s.Repo.CheckProjectExists(ctx,manifest.ExternalID)
		if exists {
			continue 
		}

		// Now let's launch a new Transaction 
		transactionError := s.Repo.ExecuteTx(ctx, func(txRepo ports.IngestionRepository)error {
			// we first create the project 
			projID, projectCreationError := txRepo.CreateProject(ctx,manifest)
			if projectCreationError != nil {
				return fmt.Errorf("an error occured while trying to create a new project from the manifest")
			}
			//we then extract the pdf from the zip and send them to MiniO

			uploadingZipFileError := s.UploadFileFromZip(ctx,zipReader,manifest.FichierDCE,projID,"DCE",txRepo)
			if uploadingZipFileError != nil {
				return fmt.Errorf("an error occured while trying to upload all of the zip files to the bucket")
			}

			// we then create the Job 
			jobID,jobCreationError := txRepo.CreateJob(ctx,projID)
			if jobCreationError != nil {
				return fmt.Errorf("an error occured while trying to create a new job")
			}
			// we then publish it into rabbitmq 
			return s.Queue.PublishJob(ctx,jobID,projID)
		})
		if transactionError != nil {
			return fmt.Errorf("Failed to process project %s:%v\n",manifest.ExternalID,transactionError)
		}
	}
	return nil
}