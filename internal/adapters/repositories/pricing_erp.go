package repositories 


import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	request, _ := http.NewRequestWithContext(ctx,"GET",fmt.Sprintf("%s/articles/%s/prix",e.ApiUrl,codeArticle),nil)

	response, responseError := e.Client.Do(request)
	if responseError != nil || response.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to call ERP pour %s : %v",codeArticle,responseError)
	}
	defer response.Body.Close()

	var result struct {
		Prix float64 `json:"prix"`
	}

	decodingResponseBodyError := json.NewDecoder(response.Body).Decode(&result)
	if decodingResponseBodyError != nil {
		return 0, fmt.Errorf("unable to parse the response json, an error occured : %v",decodingResponseBodyError)
	}

	return result.Prix, nil
}