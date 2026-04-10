package parser


import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"github.com/minio/minio-go/v7"
	"github.com/ledongthuc/pdf"
)

type MinioPDFParser struct {
	MinioClient *minio.Client
}

func NewMinioPDFParser(client *minio.Client) *MinioPDFParser {
	return &MinioPDFParser{MinioClient:client}
}

func (p *MinioPDFParser) FetchAndParse(ctx context.Context,minioPath string) (string,error) {
	// let's parse the minio path 
	if !strings.HasPrefix(minioPath, "minio://") {
		return "", fmt.Errorf("invalid minio path: missing 'minio://' prefix")
	}
	parts := strings.SplitN(strings.TrimPrefix(minioPath,"minio://"),"/",2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid miniO path format :%s ",minioPath)
	}
	bucket, objectName := parts[0],parts[1]

	// let's create a temporary file on the disk to avoid burning the RAM 
	tmpFile, createTemporaryFileError := os.CreateTemp("","kliops-pdf-*.pdf")
	if createTemporaryFileError != nil {
		return "",fmt.Errorf("failed to create temp file: %v",createTemporaryFileError)
	}
	tmpFilePath := tmpFile.Name() 

	defer func(){
		tmpFile.Close()
		os.Remove(tmpFilePath)
	}()

	obj, streamingMinioObjectErr := p.MinioClient.GetObject(ctx,bucket,objectName,minio.GetObjectOptions{})
	if streamingMinioObjectErr != nil {
		return "", fmt.Errorf("failed to fetch object from MiniO: %w",streamingMinioObjectErr)
	}
	defer obj.Close()

	_, copyingObjectToTempFileErr := io.Copy(tmpFile,obj)

	if copyingObjectToTempFileErr != nil {
		return "",fmt.Errorf("failed to stream object to disk: %v",copyingObjectToTempFileErr)
	} 

	syncError := tmpFile.Sync()
	if syncError != nil {
		return "",fmt.Errorf("failed to sync the temp file : %v",syncError)
	}

	return extractTextFromPDF(tmpFilePath)
}



func extractTextFromPDF(filePath string) (string,error) {
	f, r, openingPdfFileerr := pdf.Open(filePath)
	if openingPdfFileerr != nil {
		return "", fmt.Errorf("failed to open pdf for parsing. %s",openingPdfFileerr)
	}
	defer f.Close()

	var builder strings.Builder 

	totalPage := r.NumPage()

	for pageIndex := 1; pageIndex <= totalPage; pageIndex++{
		p := r.Page(pageIndex)

		if p.V.IsNull() {
			continue
		}
		text , err := p.GetPlainText(nil)
		if err != nil {
			fmt.Printf("Warning: failed to extract text from page %d : %v\n",pageIndex,err) 
			continue
		}
		builder.WriteString(text)
		builder.WriteString("\n")
	}
	return builder.String(),nil
}