package ports 

import (
	"context"
	"io"
)


type DocumentGenerator interface {
	// GenerateFromStream takes a flux , upload it , convert it to doc format, replace the placeholders and returns the document url
	GenerateFromStream(ctx context.Context, templatStream io.Reader,fileName string, variables map[string]string) (docID,docURL string, err error)
	// ShareWithUser takes a user email, an share the specified document with the user
	ShareWithUser(ctx context.Context,docID string,userEmail string) error 
}