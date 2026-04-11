package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/jackc/pgx/v5/pgxpool"
	
	"github.com/MiltonJ23/Kliops/internal/adapters/handlers"
	"github.com/MiltonJ23/Kliops/internal/adapters/queue"
	"github.com/MiltonJ23/Kliops/internal/adapters/repositories"
	"github.com/MiltonJ23/Kliops/internal/core/services"
)

type HealthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, relying on system environment variables")
	}

	log.Println("--> Initializing Kliops API Gateway...")

	// 1. MinIO Initialization
	minioEndpoint := os.Getenv("MINIO_ENDPOINT")
	if minioEndpoint == "" {
		minioEndpoint = "localhost:9000"
	}
	minioUseSSL := false
	if strings.HasPrefix(minioEndpoint, "https://") {
		minioUseSSL = true
		minioEndpoint = strings.TrimPrefix(minioEndpoint, "https://")
	} else if strings.HasPrefix(minioEndpoint, "http://") {
		minioEndpoint = strings.TrimPrefix(minioEndpoint, "http://")
	} else if os.Getenv("MINIO_USE_SSL") == "true" {
		minioUseSSL = true
	}

	minioRootUser := os.Getenv("MINIO_ROOT_USER")
	if minioRootUser == "" {
		log.Fatal("MINIO_ROOT_USER is required")
	}

	minioRootPassword := os.Getenv("MINIO_ROOT_PASSWORD")
	if minioRootPassword == "" {
		log.Fatal("MINIO_ROOT_PASSWORD is required")
	}

	minioStorage, err := repositories.NewMinioStorage(minioEndpoint, minioRootUser, minioRootPassword, minioUseSSL)
	if err != nil {
		log.Fatalf("Failed to initialize MinIO client: %v", err)
	}

	// 2. PostgreSQL Initialization
	dbDSN := os.Getenv("DB_DSN")
	
	dbPool, err := pgxpool.New(context.Background(), dbDSN)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer dbPool.Close()

	ingestionRepo := repositories.NewIngestionPostgres(dbPool)

	// 3. RabbitMQ Initialization
	rabbitURI := os.Getenv("RABBITMQ_URI")
	
	rabbitMQ, err := queue.NewRabbitMQAdapter(rabbitURI)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitMQ.Conn.Close()
	defer rabbitMQ.Channel.Close()

	// 4. Services Initialization
	pricingService := services.NewPricingService()
	archiveService := services.NewArchiveService(ingestionRepo, minioStorage, rabbitMQ)

	// 5. Handlers Initialization
	gatewayHandler := handlers.NewGatewayHandler(minioStorage, pricingService)
	ingestionHandler := handlers.NewIngestionHandler(archiveService, minioStorage)

	// 6. Routing
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(HealthResponse{
			Status:  "OK",
			Message: "Kliops API Gateway is running",
		})
	})

	apiMux := http.NewServeMux()
	
	// Gateway Routes
	apiMux.HandleFunc("POST /upload", gatewayHandler.HandleUpload)
	apiMux.HandleFunc("GET /price", gatewayHandler.HandlePrice)

	// Ingestion Routes
	apiMux.HandleFunc("POST /ingest/archive", ingestionHandler.UploadArchiveZip)
	apiMux.HandleFunc("POST /ingest/mercuriale", ingestionHandler.UploadMercuriale)
	apiMux.HandleFunc("POST /ingest/template", ingestionHandler.UploadTemplateDocx)

	mux.Handle("/api/v1/", handlers.APIKeyMiddleware(apiMux))

	// 7. Server Configuration
	srv := &http.Server{
		Addr:         ":8070",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Println("Starting Kliops server on port 8070...")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// 8. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down the server safely, hang on...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting gracefully. Goodbye.")
}