package repositories 


import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

type ERPPricing struct {
	ApiUrl string 
	Client *http.Client
}

func NewERPPricing(apiUrl string) *ERPPricing {
	return &ERPPricing{ApiUrl:apiUrl,Client: &http.Client{Timeout: 5 * time.Second}}
}


func (e *ERPPricing) GetPrice(ctx context.Context, codeArticle string) (float64,error) {
	encodedCode := url.PathEscape(codeArticle)
	request, err := http.NewRequestWithContext(ctx,"GET",fmt.Sprintf("%s/articles/%s/prix",e.ApiUrl,encodedCode),nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request for article %s: %w", codeArticle, err)
	}

	response, responseError := e.Client.Do(request)
	if response != nil {
		defer response.Body.Close()
	}
	
	if responseError != nil {
		return 0, fmt.Errorf("failed to call ERP for %s : %w", codeArticle, responseError)
	}
	
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		truncatedBody := string(body)
		if len(truncatedBody) > 200 {
			truncatedBody = truncatedBody[:200] + "...[truncated]"
		}
		log.Printf("ERP returned status %d for article %s: %s", response.StatusCode, codeArticle, string(body))
		return 0, fmt.Errorf("ERP returned status %d for article %s: %s", response.StatusCode, codeArticle, truncatedBody)
	}

	var result struct {
		Prix float64 `json:"prix"`
	}

	decodingResponseBodyError := json.NewDecoder(response.Body).Decode(&result)
	if decodingResponseBodyError != nil {
		return 0, fmt.Errorf("unable to parse the response json, an error occurred : %w",decodingResponseBodyError)
	}

	return result.Prix, nil
}