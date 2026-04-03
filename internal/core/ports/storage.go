package ports 

import (
	"context"
	"io"
)


// FileStorage is the contract that defines how the system interact with Blob Storage(miniO/s3)
type FileStorage interface {
	// Upload save a file in a bucket and return it's url 
	Upload(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, contentType string) (string,error)
}