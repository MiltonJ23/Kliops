package ports 

import (
	"context"
	"io"
)


// FileStorage is the contract that defines how the system interacts with Blob Storage (MinIO/S3)
type FileStorage interface {
	// Upload saves a file in a bucket and returns its URL
	Upload(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, contentType string) (string, error)
	// Delete removes a file from the bucket
	Delete(ctx context.Context, bucketName, objectName string) error
	//DownloadStream will allow me to stream large document as through a pipe to avoid clogging the RAM 
	DownloadStream(ctx context.Context,bucketName,objectName string) (io.ReadCloser,error)
}