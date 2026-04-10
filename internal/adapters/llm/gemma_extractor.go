package llm 

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"github.com/MiltonJ23/Kliops/internal/core/ports"
	"math"
	"sort"
)


type GemmaExtractor struct {
	Client *http.Client
	Model string 
	BaseUrl string
	Embedder ports.Embedder
}

func NewGemmaExtractor(baseUrl,model string,embedder ports.Embedder) *GemmaExtractor {
	return &GemmaExtractor{
		Client: &http.Client{},
		Model:model,
		BaseUrl: baseUrl,
		Embedder:embedder,
	}
}

type memoVector struct {
	Text string 
	Vector []float32
}

func cosineSimilarity(a,b []float32) float64 {
	var dotProduct,normA,normB float64 
	for i:= 0 ; i<len(a) && i<len(b); i++ {
		dotProduct += float64(a[i]*b[i])
		normA += float64(a[i]*a[i])
		normB += float64(b[i]*b[i]) 
	}

	if normA==0 || normB == 0 {
		return 0
	}
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

func getTopKSimilar(query []float32,corpus []memoVector,k int) []string {
	if len(corpus) == 0 {
		return nil
	}
	type scoredChunk struct {
		Text string 
		Score float64
	}
	var scores []scoredChunk
	for _, doc := range corpus {

		sim := cosineSimilarity(query,doc.Vector)
		scores = append(scores,scoredChunk{Text:doc.Text,Score:sim})
	}
	sort.Slice(scores, func(i, j int) bool { 
		return scores[i].Score > scores[j].Score 
	})
	var topK []string
	limit := k
	if len(scores) < k { limit = len(scores) }
	for i := 0; i < limit; i++ {
		if scores[i].Score > 0.40 { 
			topK = append(topK, scores[i].Text)
		}
	}
	return topK
}

// chunkText splits big text into a slice of chunked text to be ingested better given the limitation of models
func chunkText(text string,chunkSize int) []string {
	runes := []rune(text)
	if len(runes) == 0 {
		return []string{}
	}

	var chunks []string 

	for i:=0;i<len(runes);i+=chunkSize {
		end := i + chunkSize 
		if end > len(runes) {
			end = len(runes)
		}
		chunks  = append(chunks,string(runes[i:end]))
	}
	return chunks
}


func (g *GemmaExtractor) ExtractExigenceReponsePairFromChunkUsingGemma(ctx context.Context,dceChunk,memChunk string) ([]ports.ExtractedPair,error) {
	prompt := fmt.Sprintf(`Tu es un ingénieur BTP Chevronne ayant fait ses classes a l'X cursus Genie Civil, petri de tes 20 annees d'experience au sein de la boite de BTP francaise VINCI. Analyse CETTE PORTION du Cahier des Clauses Techniques (DCE) et CETTE PORTION du Mémoire Technique.
Identifie les exigences techniques majeures dans le DCE et trouve la réponse correspondante dans le Mémoire.
Tu DOIS répondre UNIQUEMENT avec un tableau JSON valide respectant cette structure exacte :
[{"exigence": "texte de l'exigence", "reponse": "réponse de l'entreprise"}]
Si tu ne trouves aucune exigence, renvoie [].

DCE (Extrait) : %s
MEMOIRE (Extrait) : %s`, dceChunk, memChunk)

	reqBody := map[string]interface{}{
		"model": g.Model,
		"prompt": prompt,
		"format":"json",
		"stream":false,
	}

	jsonData, marshalingRequestBodyErr := json.Marshal(reqBody)
	if marshalingRequestBodyErr != nil {
		return nil, fmt.Errorf("Failed to marshal the request body: %v",marshalingRequestBodyErr)
	}

	request, buildingRequestErr := http.NewRequestWithContext(ctx,"POST",fmt.Sprintf("%s/api/generate",g.BaseUrl),bytes.NewBuffer(jsonData))
	if buildingRequestErr != nil {
		return nil, fmt.Errorf("failed to building the request: %v",buildingRequestErr)
	}
	request.Header.Set("Content-type","application/json")

	response, reponseErr := g.Client.Do(request)
	if reponseErr != nil {
		return nil, fmt.Errorf("Ollama call faile: %v",reponseErr)
	}
	defer response.Body.Close() 

	if response.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("Ollama status %d : %s",response.StatusCode,string(bodyBytes))
	}

	var result struct {
		Reponse string `json:"reponse"`
	}

	decodingResponseJsonErr := json.NewDecoder(response.Body).Decode(&result)
	if decodingResponseJsonErr != nil {
		return nil, fmt.Errorf("an error occured while decoding the response from ollama: %v",decodingResponseJsonErr)
	}


	// now let's clean the response from the model 
	cleanJSON := strings.TrimSpace(result.Reponse)
	cleanJSON = strings.TrimPrefix(cleanJSON, "```json")
	cleanJSON = strings.TrimPrefix(cleanJSON, "```")
	cleanJSON = strings.TrimSuffix(cleanJSON, "```")
	cleanJSON = strings.TrimSpace(cleanJSON)

	if cleanJSON == "" || cleanJSON == "[]" {
		return []ports.ExtractedPair{}, nil
	}

	var pairs []ports.ExtractedPair 
	unmarshalingCleanJSonErr := json.Unmarshal([]byte(cleanJSON),&pairs)
	if unmarshalingCleanJSonErr != nil {
		return nil, fmt.Errorf("failed to unmarshall the cleaned Json response: %v",unmarshalingCleanJSonErr)
	}

		return pairs, nil
}


func (g *GemmaExtractor) ExtractRequirementsAndAnswers(ctx context.Context, dceText, memoireText string) ([]ports.ExtractedPair, error) {
	dceChunks := chunkText(dceText, 2500)
	memChunks := chunkText(memoireText, 2500)

	var memoKnowledge []memoVector
	for _, mChunk := range memChunks {
		vec, err := g.Embedder.CreateEmbedding(ctx, mChunk)
		if err != nil {
			fmt.Printf("[Embedder] Warning: failed to embed memo chunk: %v\n", err)
			continue
		}
		memoKnowledge = append(memoKnowledge, memoVector{Text: mChunk, Vector: vec})
	}

	var allPairs []ports.ExtractedPair 
	for i, dChunk := range dceChunks {
		dVec, err := g.Embedder.CreateEmbedding(ctx, dChunk)
		if err != nil {
			fmt.Printf("[Embedder] Warning: failed to embed DCE chunk %d: %v\n", i, err)
			continue
		}

		topKMemoTexts := getTopKSimilar(dVec, memoKnowledge, 3)
		combinedContext := strings.Join(topKMemoTexts, "\n\n[...]\n\n")
		if combinedContext == "" {
			combinedContext = "Aucune information sémantiquement proche trouvée dans le mémoire."
		}
		pairs, err := g.ExtractExigenceReponsePairFromChunkUsingGemma(ctx, dChunk, combinedContext)
		if err != nil {
			fmt.Printf("[Gemma] Warning: chunk %d failed to process: %v\n", i, err)
			continue
		}
		allPairs = append(allPairs, pairs...)
	}

	return allPairs, nil
}