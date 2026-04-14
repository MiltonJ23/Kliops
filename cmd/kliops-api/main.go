package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	agentadapter "github.com/MiltonJ23/Kliops/internal/adapters/agent"
	googleworkspace "github.com/MiltonJ23/Kliops/internal/adapters/google_workspace"
	"github.com/MiltonJ23/Kliops/internal/adapters/handlers"
	"github.com/MiltonJ23/Kliops/internal/adapters/llm"
	"github.com/MiltonJ23/Kliops/internal/adapters/parser"
	"github.com/MiltonJ23/Kliops/internal/adapters/queue"
	"github.com/MiltonJ23/Kliops/internal/adapters/repositories"
	"github.com/MiltonJ23/Kliops/internal/core/services"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"google.golang.org/grpc/connectivity"
)

var version = "dev"

type appConfig struct {
	Addr                  string
	DatabaseURL           string
	MinioEndpoint         string
	MinioRootUser         string
	MinioRootPassword     string
	MinioUseSSL           bool
	RabbitMQURL           string
	QdrantAddr            string
	QdrantCollection      string
	OllamaBaseURL         string
	OllamaChatModel       string
	OllamaEmbeddingModel  string
	GoogleCredentialsFile string
	PricingExcelPath      string
	ERPBaseURL            string
	WorkerConcurrency     int
}

type application struct {
	server     *http.Server
	dbPool     *pgxpool.Pool
	mqAdapter  *queue.RabbitMQAdapter
	vectorRepo *repositories.QdrantRepository
	minio      *repositories.MinioStorage
	ollamaURL  string
	startedAt  time.Time
}

type readinessCheck struct {
	Name  string `json:"name"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Info: Fichier .env absent, utilisation des variables système.")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("configuration invalide: %v", err)
	}

	app, err := newApplication(ctx, cfg)
	if err != nil {
		log.Fatalf("échec de démarrage: %v", err)
	}
	defer app.close()

	log.Printf("--> Démarrage de Kliops API Gateway (addr=%s version=%s model=%s)", cfg.Addr, version, cfg.OllamaChatModel)

	go func() {
		log.Printf("Serveur écoute sur %s", app.server.Addr)
		if err := app.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Erreur serveur: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Arrêt gracieux en cours...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Erreur lors de l'arrêt du serveur: %v", err)
	}

	log.Println("Kliops arrêté proprement.")
}

func loadConfig() (appConfig, error) {
	cfg := appConfig{
		Addr:                  envOrDefault("APP_ADDR", ":8070"),
		DatabaseURL:           strings.TrimSpace(os.Getenv("DATABASE_URL")),
		MinioEndpoint:         strings.TrimSpace(os.Getenv("MINIO_ENDPOINT")),
		MinioRootUser:         strings.TrimSpace(os.Getenv("MINIO_ROOT_USER")),
		MinioRootPassword:     strings.TrimSpace(os.Getenv("MINIO_ROOT_PASSWORD")),
		MinioUseSSL:           strings.EqualFold(strings.TrimSpace(os.Getenv("MINIO_USE_SSL")), "true"),
		RabbitMQURL:           strings.TrimSpace(os.Getenv("RABBITMQ_URL")),
		QdrantAddr:            strings.TrimSpace(os.Getenv("QDRANT_ADDR")),
		QdrantCollection:      envOrDefault("QDRANT_COLLECTION", "btp_knowledge"),
		OllamaBaseURL:         strings.TrimRight(envOrDefault("OLLAMA_BASE_URL", "http://localhost:11434"), "/"),
		OllamaChatModel:       envOrDefault("OLLAMA_CHAT_MODEL", "gemma4:e4b"),
		OllamaEmbeddingModel:  envOrDefault("OLLAMA_EMBED_MODEL", "mxbai-embed-large"),
		GoogleCredentialsFile: envOrDefault("GOOGLE_CREDENTIALS_FILE", "./credentials.json"),
		PricingExcelPath:      envOrDefault("PRICING_EXCEL_PATH", "./dummy_prices.xlsx"),
		ERPBaseURL:            strings.TrimRight(strings.TrimSpace(os.Getenv("ERP_BASE_URL")), "/"),
		WorkerConcurrency:     4,
	}

	required := map[string]string{
		"DATABASE_URL":        cfg.DatabaseURL,
		"MINIO_ENDPOINT":      cfg.MinioEndpoint,
		"MINIO_ROOT_USER":     cfg.MinioRootUser,
		"MINIO_ROOT_PASSWORD": cfg.MinioRootPassword,
		"RABBITMQ_URL":        cfg.RabbitMQURL,
		"QDRANT_ADDR":         cfg.QdrantAddr,
		"API_KEY_SECRET":      strings.TrimSpace(os.Getenv("API_KEY_SECRET")),
	}
	for name, value := range required {
		if value == "" {
			return appConfig{}, fmt.Errorf("%s est requis", name)
		}
	}

	return cfg, nil
}

func newApplication(ctx context.Context, cfg appConfig) (*application, error) {
	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("impossible de connecter Postgres: %w", err)
	}
	if err := dbPool.Ping(ctx); err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("postgres indisponible: %w", err)
	}

	storage, err := repositories.NewMinioStorage(cfg.MinioEndpoint, cfg.MinioRootUser, cfg.MinioRootPassword, cfg.MinioUseSSL)
	if err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("échec MinIO: %w", err)
	}

	mqAdapter, err := queue.NewRabbitMQAdapter(cfg.RabbitMQURL)
	if err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("échec RabbitMQ: %w", err)
	}

	embedder := llm.NewOllamaEmbedder(cfg.OllamaBaseURL, cfg.OllamaEmbeddingModel)
	gemma := llm.NewGemmaExtractor(cfg.OllamaBaseURL, cfg.OllamaChatModel, embedder)

	vectorRepo, err := repositories.NewQdrantRepository(cfg.QdrantAddr, embedder, cfg.QdrantCollection)
	if err != nil {
		_ = mqAdapter.Close()
		dbPool.Close()
		return nil, fmt.Errorf("échec Qdrant: %w", err)
	}

	ingestionRepo := repositories.NewIngestionPostgres(dbPool)
	archiveService := services.NewArchiveService(ingestionRepo, storage, mqAdapter)
	knowledgeService := services.NewKnowledgeService(vectorRepo, ingestionRepo)
	pricingService := services.NewPricingService()
	pricingService.RegisterStrategy("postgres", repositories.NewPostgresPricing(dbPool))
	if fileExists(cfg.PricingExcelPath) {
		pricingService.RegisterStrategy("excel", repositories.NewExcelPricing(cfg.PricingExcelPath))
	} else {
		log.Printf("Warning: fichier Excel de pricing introuvable, stratégie 'excel' désactivée (%s)", cfg.PricingExcelPath)
	}
	if cfg.ERPBaseURL != "" {
		pricingService.RegisterStrategy("erp", repositories.NewERPPricing(cfg.ERPBaseURL))
	}

	workspaceAdapter, err := googleworkspace.NewWorkspaceAdapter(ctx, cfg.GoogleCredentialsFile)
	if err != nil {
		_ = vectorRepo.Close()
		_ = mqAdapter.Close()
		dbPool.Close()
		return nil, fmt.Errorf("échec Google Workspace: %w", err)
	}
	documentService := services.NewDocumentService(storage, workspaceAdapter)
	orchestrator := agentadapter.NewKliopsOrchestratorWithModel(cfg.OllamaBaseURL, cfg.OllamaChatModel, knowledgeService, pricingService, documentService)
	agentService := services.NewAgentService(orchestrator)

	pdfParser := parser.NewMinioPDFParser(storage.Client)
	workerService := services.NewWorkerService(ingestionRepo, gemma, vectorRepo, pdfParser)
	if err := mqAdapter.ConsumeJob(ctx, cfg.WorkerConcurrency, workerService.HandleJob); err != nil {
		_ = vectorRepo.Close()
		_ = mqAdapter.Close()
		dbPool.Close()
		return nil, fmt.Errorf("échec lancement worker RabbitMQ: %w", err)
	}

	gatewayHandler := handlers.NewGatewayHandler(storage, pricingService)
	ingestionHandler := handlers.NewIngestionHandler(archiveService, storage)
	agentHandler := handlers.NewAgentHandler(agentService)

	app := &application{
		dbPool:     dbPool,
		mqAdapter:  mqAdapter,
		vectorRepo: vectorRepo,
		minio:      storage,
		ollamaURL:  cfg.OllamaBaseURL,
		startedAt:  time.Now().UTC(),
	}
	app.server = &http.Server{
		Addr:         cfg.Addr,
		Handler:      recoveryMiddleware(app.routes(gatewayHandler, ingestionHandler, agentHandler)),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return app, nil
}

func (a *application) routes(gatewayHandler *handlers.GatewayHandler, ingestionHandler *handlers.IngestionHandler, agentHandler *handlers.AgentHandler) http.Handler {
	mux := http.NewServeMux()
	apiMux := http.NewServeMux()

	apiMux.HandleFunc("POST /ingest/archive", ingestionHandler.UploadArchiveZip)
	apiMux.HandleFunc("POST /ingest/mercuriale", ingestionHandler.UploadMercuriale)
	apiMux.HandleFunc("POST /ingest/template", ingestionHandler.UploadTemplateDocx)
	apiMux.HandleFunc("POST /upload", gatewayHandler.HandleUpload)
	apiMux.HandleFunc("GET /price", gatewayHandler.HandlePrice)
	apiMux.HandleFunc("POST /agent/ask", agentHandler.HandleQuery)

	mux.HandleFunc("GET /{$}", a.handleRoot)
	mux.HandleFunc("GET /health", a.handleLiveness)
	mux.HandleFunc("GET /livez", a.handleLiveness)
	mux.HandleFunc("GET /readyz", a.handleReadiness)
	mux.HandleFunc("GET /version", a.handleVersion)
	mux.Handle("/api/v1/", handlers.APIKeyMiddleware(http.StripPrefix("/api/v1", apiMux)))

	return mux
}

func (a *application) handleRoot(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"service":    "kliops-api",
		"version":    version,
		"started_at": a.startedAt.Format(time.RFC3339),
		"docs_hint":  "/api/v1/* avec X-API-KEY",
	})
}

func (a *application) handleLiveness(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "UP"})
}

func (a *application) handleVersion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"version": version})
}

func (a *application) handleReadiness(w http.ResponseWriter, r *http.Request) {
	checks := []readinessCheck{
		a.checkPostgres(r.Context()),
		a.checkMinIO(r.Context()),
		a.checkRabbitMQ(),
		a.checkQdrant(),
		a.checkOllama(r.Context()),
	}

	status := http.StatusOK
	for _, check := range checks {
		if !check.OK {
			status = http.StatusServiceUnavailable
			break
		}
	}

	writeJSON(w, status, map[string]any{
		"status": statusText(status),
		"checks": checks,
	})
}

func (a *application) checkPostgres(ctx context.Context) readinessCheck {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := a.dbPool.Ping(ctx); err != nil {
		return readinessCheck{Name: "postgres", OK: false, Error: err.Error()}
	}
	return readinessCheck{Name: "postgres", OK: true}
}

func (a *application) checkMinIO(ctx context.Context) readinessCheck {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if _, err := a.minio.Client.ListBuckets(ctx); err != nil {
		return readinessCheck{Name: "minio", OK: false, Error: err.Error()}
	}
	return readinessCheck{Name: "minio", OK: true}
}

func (a *application) checkRabbitMQ() readinessCheck {
	if a.mqAdapter == nil || a.mqAdapter.Conn == nil || a.mqAdapter.Conn.IsClosed() {
		return readinessCheck{Name: "rabbitmq", OK: false, Error: "connexion fermée"}
	}
	return readinessCheck{Name: "rabbitmq", OK: true}
}

func (a *application) checkQdrant() readinessCheck {
	if a.vectorRepo == nil || a.vectorRepo.Conn == nil {
		return readinessCheck{Name: "qdrant", OK: false, Error: "connexion absente"}
	}
	if state := a.vectorRepo.Conn.GetState(); state == connectivity.Shutdown {
		return readinessCheck{Name: "qdrant", OK: false, Error: state.String()}
	}
	return readinessCheck{Name: "qdrant", OK: true}
}

func (a *application) checkOllama(ctx context.Context) readinessCheck {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.ollamaURL+"/api/tags", nil)
	if err != nil {
		return readinessCheck{Name: "ollama", OK: false, Error: err.Error()}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return readinessCheck{Name: "ollama", OK: false, Error: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return readinessCheck{Name: "ollama", OK: false, Error: resp.Status}
	}
	return readinessCheck{Name: "ollama", OK: true}
}

func (a *application) close() {
	if a.vectorRepo != nil {
		if err := a.vectorRepo.Close(); err != nil {
			log.Printf("Erreur fermeture Qdrant: %v", err)
		}
	}
	if a.mqAdapter != nil {
		if err := a.mqAdapter.Close(); err != nil {
			log.Printf("Erreur fermeture RabbitMQ: %v", err)
		}
	}
	if a.dbPool != nil {
		a.dbPool.Close()
	}
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic interceptée sur %s %s: %v", r.Method, r.URL.Path, recovered)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("Erreur encodage réponse JSON: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func fileExists(path string) bool {
	info, err := os.Stat(filepath.Clean(path))
	return err == nil && !info.IsDir()
}

func statusText(status int) string {
	if status == http.StatusOK {
		return "READY"
	}
	return "DEGRADED"
}
