package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	openai "github.com/sashabaranov/go-openai"

	"github.com/MiltonJ23/Kliops/internal/core/services"
)

// maxToolIterations caps the agentic loop to prevent infinite LLM cycles.
const maxToolIterations = 12

// llmModel is the Ollama model tag. Gemma 4 via Ollama uses the tag as-is.
const llmModel = "gemma4:e4b"

// ─── Tool argument types ──────────────────────────────────────────────────────

// KnowledgeArgs is the input schema for the search_knowledge tool.
type KnowledgeArgs struct {
	Query string `json:"query"`
}

// PricingArgs is the input schema for the get_price tool.
type PricingArgs struct {
	CodeArticle string `json:"code_article"`
	Source      string `json:"source"` // "excel" | "erp"
}

// DocumentArgs is the input schema for the generate_document tool.
type DocumentArgs struct {
	ProjectName string            `json:"project_name"`
	TargetEmail string            `json:"target_email"`
	Variables   map[string]string `json:"variables"`
}

// ─── Orchestrator ─────────────────────────────────────────────────────────────

// KliopsOrchestrator implements services.AgentOrchestrator.
// It drives a tool-calling loop using Gemma 4 via Ollama's OpenAI-compatible
// endpoint, routing each tool call to the appropriate domain service.
type KliopsOrchestrator struct {
	client    *openai.Client
	model     string
	knowledge *services.KnowledgeService
	pricing   *services.PricingService
	document  *services.DocumentService
}

// NewKliopsOrchestrator creates a production-ready orchestrator.
// ollamaBaseURL should be e.g. "http://localhost:11434" (no trailing slash).
func NewKliopsOrchestrator(
	ollamaBaseURL string,
	knowledge *services.KnowledgeService,
	pricing *services.PricingService,
	document *services.DocumentService,
) *KliopsOrchestrator {
	return NewKliopsOrchestratorWithModel(ollamaBaseURL, llmModel, knowledge, pricing, document)
}

func NewKliopsOrchestratorWithModel(
	ollamaBaseURL string,
	model string,
	knowledge *services.KnowledgeService,
	pricing *services.PricingService,
	document *services.DocumentService,
) *KliopsOrchestrator {
	if model == "" {
		model = llmModel
	}
	cfg := openai.DefaultConfig("ollama") // Ollama ignores the API key value
	cfg.BaseURL = ollamaBaseURL + "/v1"

	return &KliopsOrchestrator{
		client:    openai.NewClientWithConfig(cfg),
		model:     model,
		knowledge: knowledge,
		pricing:   pricing,
		document:  document,
	}
}

// ─── Tool definitions ─────────────────────────────────────────────────────────

func (o *KliopsOrchestrator) toolDefinitions() []openai.Tool {
	return []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "search_knowledge",
				Description: "Recherche dans la base vectorielle les réponses techniques validées sur des projets passés pour une exigence CCTP donnée.",
				Parameters: schema(
					props{
						"query": strProp("L'exigence technique du CCTP à rechercher dans l'historique."),
					},
					"query",
				),
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_price",
				Description: "Retourne le prix unitaire HT (en FCFA) d'un article depuis la mercuriale Excel ou l'ERP interne.",
				Parameters: schema(
					props{
						"code_article": strProp("Le code article exact à interroger."),
						"source":       enumProp([]string{"excel", "erp"}, "La source tarifaire à consulter."),
					},
					"code_article", "source",
				),
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "generate_document",
				Description: "Génère le mémoire technique DOCX final à partir du template SITINFRA, substitue tous les placeholders, et partage le document par email. À appeler UNE SEULE FOIS, lorsque toutes les données sont collectées.",
				Parameters: schema(
					props{
						"project_name": strProp("Nom du projet pour le titre du document."),
						"target_email": strProp("Adresse email du destinataire du document généré."),
						"variables":    objectProp("Map clé→valeur des placeholders du template SITINFRA. Tous les placeholders identifiés dans le template doivent être couverts."),
					},
					"project_name", "target_email", "variables",
				),
			},
		},
	}
}

// ─── Tool execution ───────────────────────────────────────────────────────────

// executeTool routes an LLM tool call to the appropriate domain service.
// It never returns a Go error — execution errors are encoded as JSON so the
// LLM can reason about them and decide how to proceed.
func (o *KliopsOrchestrator) executeTool(ctx context.Context, name, argsJSON string) string {
	switch name {

	case "search_knowledge":
		var args KnowledgeArgs
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return errJSON("arguments invalides pour search_knowledge: " + err.Error())
		}
		result, err := o.knowledge.RetrieveRelevantMethodologies(ctx, args.Query)
		if err != nil {
			return errJSON("échec de la recherche vectorielle: " + err.Error())
		}
		return result

	case "get_price":
		var args PricingArgs
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return errJSON("arguments invalides pour get_price: " + err.Error())
		}
		price, err := o.pricing.GetPrice(ctx, args.Source, args.CodeArticle)
		if err != nil {
			return errJSON(fmt.Sprintf("prix introuvable pour '%s' dans '%s': %v", args.CodeArticle, args.Source, err))
		}
		out, _ := json.Marshal(map[string]any{
			"code_article": args.CodeArticle,
			"source":       args.Source,
			"prix_ht_fcfa": price,
		})
		return string(out)

	case "generate_document":
		var args DocumentArgs
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return errJSON("arguments invalides pour generate_document: " + err.Error())
		}
		url, err := o.document.CompileTechnicalMemory(ctx, args.ProjectName, args.Variables, args.TargetEmail)
		if err != nil {
			return errJSON("génération du document échouée: " + err.Error())
		}
		out, _ := json.Marshal(map[string]string{"document_url": url})
		return string(out)

	default:
		return errJSON("outil inconnu: " + name)
	}
}

// ─── Agentic loop ─────────────────────────────────────────────────────────────

const supervisorSystemPrompt = `Tu es Kliops-Supervisor, l'agent de réponse aux appels d'offres de SITINFRA.

Procédure OBLIGATOIRE — tu dois la suivre intégralement dans cet ordre :

ÉTAPE 1 — Analyse
  Lis attentivement le contenu du DCE fourni et identifie TOUTES les exigences techniques et les lots de travaux.

ÉTAPE 2 — Recherche documentaire
  Pour chaque exigence technique identifiée, appelle "search_knowledge" afin de récupérer les réponses validées sur des projets passés similaires.

ÉTAPE 3 — Chiffrage
  Pour chaque matériau, prestation ou équipement nécessitant un prix, appelle "get_price" avec le code article et la source appropriée.
  Ne suppose JAMAIS un prix. Utilise TOUJOURS cet outil.

ÉTAPE 4 — Génération du document
  Une fois que tu as collecté TOUTES les informations (réponses techniques + prix), appelle "generate_document" UNE SEULE FOIS.
  Remplis TOUS les placeholders du template SITINFRA. Si une information est vraiment introuvable, affecte la valeur "À COMPLÉTER".

ÉTAPE 5 — Réponse finale
  Ta seule réponse finale à l'utilisateur est l'URL du document généré, rien d'autre.

Règles absolues :
  - Ne génère JAMAIS de prix par estimation.
  - N'appelle "generate_document" qu'une seule fois, à la fin.
  - Ne réponds pas à l'utilisateur avant que le document soit généré.`

// Run drives the full agentic tool-calling loop for a given prompt.
// It implements services.AgentOrchestrator.
func (o *KliopsOrchestrator) Run(ctx context.Context, prompt string) (string, error) {
	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: supervisorSystemPrompt},
		{Role: openai.ChatMessageRoleUser, Content: prompt},
	}
	tools := o.toolDefinitions()

	for i := 0; i < maxToolIterations; i++ {
		resp, err := o.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:    o.model,
			Messages: messages,
			Tools:    tools,
		})
		if err != nil {
			return "", fmt.Errorf("itération %d — appel LLM échoué: %w", i, err)
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("itération %d — réponse LLM vide", i)
		}

		assistantMsg := resp.Choices[0].Message

		// Always append the assistant turn to maintain conversation coherence.
		messages = append(messages, assistantMsg)

		// No tool calls → the LLM produced its final answer.
		if len(assistantMsg.ToolCalls) == 0 {
			if assistantMsg.Content == "" {
				return "", fmt.Errorf("itération %d — réponse finale vide sans appels d'outils", i)
			}
			return assistantMsg.Content, nil
		}

		// Execute every tool call and feed results back.
		for _, tc := range assistantMsg.ToolCalls {
			log.Printf("[KliopsOrchestrator] outil=%s id=%s args=%s",
				tc.Function.Name, tc.ID, tc.Function.Arguments)

			result := o.executeTool(ctx, tc.Function.Name, tc.Function.Arguments)

			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}

	return "", fmt.Errorf("l'agent n'a pas convergé après %d itérations — vérifier le prompt système ou augmenter maxToolIterations", maxToolIterations)
}

// ─── JSON schema helpers (keep tool definitions readable) ─────────────────────

type props = map[string]any

func schema(properties props, required ...string) map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

func strProp(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func enumProp(values []string, description string) map[string]any {
	return map[string]any{"type": "string", "enum": values, "description": description}
}

func objectProp(description string) map[string]any {
	return map[string]any{
		"type":                 "object",
		"description":          description,
		"additionalProperties": map[string]any{"type": "string"},
	}
}

// errJSON serialises an error message as a JSON object so the LLM can parse it.
func errJSON(msg string) string {
	out, _ := json.Marshal(map[string]string{"error": msg})
	return string(out)
}
