package llm


import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// OllamaEmbedder implements the interface ports.Embedder
type OllamaEmbedder struct {
	BaseUrl string 
	Model string 
	Client *http.Client
}

// initialize the client use to talk to my local model (running in ollama)
func NewOllamaEmbedder(baseUrl,model string) *OllamaEmbedder {
	return &OllamaEmbedder{
		BaseUrl: baseUrl,
		Model: model,
		Client: &http.Client{},
	}
}

// EmbeddingRequest refers to the json structure expectd by ollama 
type EmbeddingRequest struct {
	Model string `json:"model"`
	Prompt string `json:"prompt"`
}

type EmbeddingResponse struct {
	Embedding []float64	`json:"embedding"`
}
 
func (o *OllamaEmbedder) CreateEmbedding(ctx context.Context, text string) ([]float32,error) {
	requestBody := EmbeddingRequest{
		Model: o.Model,
		Prompt: text,
	}
	jsonData, marshallingRequestBodyError := json.Marshal(requestBody)
	if marshallingRequestBodyError != nil {
		return nil, fmt.Errorf("an error occured while marshalling the request body: %v",marshallingRequestBodyError)
	}

	// we then create the Http Request 
	url := fmt.Sprintf("%s/api/embeddings",o.BaseUrl)
	request, forgingRequestError := http.NewRequestWithContext(ctx,"POST",url,bytes.NewBuffer(jsonData))
	if forgingRequestError != nil {
		return nil, fmt.Errorf("failed to forge the http request:%v",forgingRequestError)
	}
	request.Header.Set("content-type","application/json")

	response, requestExecutionError := o.Client.Do(request)
	if requestExecutionError != nil {
		return nil, fmt.Errorf("failed to connect to Ollama, check if it is running: %v",requestExecutionError)
	}

	defer response.Body.Close() 

	if response.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("ollama send a failed http response %d :  %v",response.StatusCode,string(bodyBytes))
	}

	// now let's decode the answer from the embedding model 
	var resBody EmbeddingResponse 
	unmarshallingResponseError := json.NewDecoder(response.Body).Decode(&resBody)
	if unmarshallingResponseError != nil {
		return nil, fmt.Errorf("failed to decode the response from the embedding model: %v",unmarshallingResponseError)
	}

	// now let's convert its type 
	vector := make([]float32, len(resBody.Embedding))
	for i,v := range resBody.Embedding{
		vector[i] = float32(v)
	}

	return vector,nil
}