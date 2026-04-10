package repositories 

import (
	"context"
	"fmt"
	"io"
	"log"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioStorage struct {
	Client *minio.Client
}


func NewMinioStorage(endPoint, accessKey,secretKey string ,useSSL bool) (*MinioStorage,error) {
	client, clientCreationError := minio.New(endPoint,&minio.Options{
		Creds: credentials.NewStaticV4(accessKey,secretKey,""),
		Secure: useSSL,
	})
	if clientCreationError != nil {
		return nil, fmt.Errorf("failed to initialize MiniO Client : %v",clientCreationError)
	}

	return &MinioStorage{Client:client},nil
}


func (m *MinioStorage) Upload(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, contentType string) (string,error) {
	// let's first of all check if the bucket exist 
	bucketCreationError := m.Client.MakeBucket(ctx,bucketName,minio.MakeBucketOptions{})
	if bucketCreationError != nil {
		exists, checkExistenceError := m.Client.BucketExists(ctx,bucketName)
		if checkExistenceError != nil {
			return "", fmt.Errorf("failed to create bucket %s and failed to check existence: %w", bucketName, checkExistenceError)
		}
		if !exists {
			return "", fmt.Errorf("failed to create bucket %s: %w", bucketName, bucketCreationError)
		}
		// else, bucket exists, proceed
	} else {
		log.Printf("Bucket %s created successfully ",bucketName)
	}

	// let's manage the upload to MiniO 
	info, uploadingError := m.Client.PutObject(ctx,bucketName,objectName,reader,objectSize,minio.PutObjectOptions{
		ContentType: contentType,
	})
	if uploadingError != nil {
		return "",fmt.Errorf("failed to upload %s in miniO : %v",objectName,uploadingError)
	}
	return fmt.Sprintf("minio://%s/%s",bucketName,info.Key),nil
}

func (m *MinioStorage) Delete(ctx context.Context, bucketName, objectName string) error {
	err := m.Client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	return err
}