package googleworkspace

import (
	"context"
	"fmt"
	"io"

	"github.com/MiltonJ23/Kliops/internal/core/ports"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/option"
)


var _ ports.DocumentGenerator =  (*WorkspaceAdapter)(nil)


type WorkspaceAdapter struct {
	DriveSvc	*drive.Service 
	DocsSvc		*docs.Service 
}


func NewWorkspaceAdapter(ctx context.Context, credentialsFilePath string) (*WorkspaceAdapter,error) {
		options := option.WithCredentialsFile(credentialsFilePath)

		driveService, driveServiceInitializationErr := drive.NewService(ctx,options)
		if driveServiceInitializationErr != nil {
			return nil, fmt.Errorf("failed to initialize the Google Drive service, ensure the Google Drive API is enabled for this service account: %v",driveServiceInitializationErr)
		}

		docService, docServiceInitializationErr := docs.NewService(ctx,options)
		if docServiceInitializationErr != nil {
			return nil, fmt.Errorf("failed to initialize the Google Docs service, ensure the Google Docs API is enabled for this service account: %v",docServiceInitializationErr)
		}

		return &WorkspaceAdapter{
		DriveSvc: driveService,
		DocsSvc: docService,
		},nil
}


func (w *WorkspaceAdapter) GenerateFromStream(ctx context.Context, templateStream io.Reader,fileName string, variables map[string]string) (docID,docURL string, err error) {
	// let's force Google Drive to convert the incoming stream to google document 
	fileMetaData := &drive.File{
		Name: fileName,
		MimeType: "application/vnd.google-apps.document",
	}

	driveFile, driveFileCreationErr := w.DriveSvc.Files.Create(fileMetaData).Media(templateStream).Context(ctx).Do()

	if driveFileCreationErr != nil {
		return "","",fmt.Errorf("failed to upload and convert document %s to Google Drive: %v",fileName,driveFileCreationErr)
	}

	if len(variables) > 0 {
		var requests []*docs.Request
		for key, val := range variables {
			requests = append(requests, &docs.Request{
				ReplaceAllText: &docs.ReplaceAllTextRequest{
					ContainsText: &docs.SubstringMatchCriteria{
						Text:      fmt.Sprintf("{{%s}}", key),
						MatchCase: true,
					},
					ReplaceText: val,
				},
			})

		}
		batchUpdateReq := &docs.BatchUpdateDocumentRequest{
			Requests: requests,
		}

		_, batchUpdatePlaceholderErr := w.DocsSvc.Documents.BatchUpdate(driveFile.Id,batchUpdateReq).Context(ctx).Do()
		if batchUpdatePlaceholderErr != nil {
			return driveFile.Id, "", fmt.Errorf("failed to inject variables via batch update: %w", batchUpdatePlaceholderErr)
		}
	}

	docUrl := fmt.Sprintf("https://docs.google.com/document/d/%s/edit", driveFile.Id) 
	return driveFile.Id,docUrl,nil
}


func (w *WorkspaceAdapter) ShareWithUser(ctx context.Context,docID string,userEmail string) error {
	permission := &drive.Permission{
		Type: "user",
		Role: "writer",
		EmailAddress: userEmail,
	}

	_, settingDriveFilePermissionAndAllowNewUserErr := w.DriveSvc.Permissions.Create(docID, permission).SendNotificationEmail(true).Context(ctx).Do()
	if settingDriveFilePermissionAndAllowNewUserErr != nil {
		return fmt.Errorf("failed to share document with %s: %w", userEmail, settingDriveFilePermissionAndAllowNewUserErr)
	}
	return nil
}