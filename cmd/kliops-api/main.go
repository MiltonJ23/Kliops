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
	"github.com/MiltonJ23/Kliops/internal/adapters/handlers"
	"github.com/MiltonJ23/Kliops/internal/adapters/repositories"
	"github.com/MiltonJ23/Kliops/internal/core/services"
)



type HealthResponse struct {
	Status string `json:"status"`
	Message string `json:"message"`
}

func main(){

	if err := godotenv.Load(); err != nil {
		log.Println("Warning : file .env not found ")
	}

	log.Println("--> Initializing Kliops API Gateway ....")



	minioEndpoint := os.Getenv("MINIO_ENDPOINT")
	if minioEndpoint == "" {
		minioEndpoint = "localhost:9000"
	}
	
	minioRootUser := os.Getenv("MINIO_ROOT_USER")
	if minioRootUser == "" {
		log.Fatal("MINIO_ROOT_USER environment variable is required and cannot be empty")
	}
	
	minioRootPassword := os.Getenv("MINIO_ROOT_PASSWORD")
	if minioRootPassword == "" {
		log.Fatal("MINIO_ROOT_PASSWORD environment variable is required and cannot be empty")
	}
	
	minioUseSSL := os.Getenv("MINIO_USE_SSL") == "true"
	if !minioUseSSL && strings.HasPrefix(minioEndpoint, "https://") {
		minioUseSSL = true
	}
	
	minioStorage, err := repositories.NewMinioStorage(minioEndpoint, minioRootUser, minioRootPassword, minioUseSSL)
	if err != nil {
		log.Fatalf("failed to initialize MinIO storage for endpoint %s: %v", minioEndpoint, err)
	}

	pricingService := services.NewPricingService()
	
	erpBaseURL := os.Getenv("ERP_BASE_URL")
	if erpBaseURL == "" {
		erpBaseURL = "http://api.erp-btp.local"
	}
	
	excelStrategy := repositories.NewExcelPricing("dummy_prices.xlsx")
	erpStrategy := repositories.NewERPPricing(erpBaseURL) 

	pricingService.RegisterStrategy("excel",excelStrategy)
	pricingService.RegisterStrategy("erp",erpStrategy)

	// we then initialize the gateway 
	gatewayHandler := handlers.NewGatewayHandler(minioStorage,pricingService)


	mux := http.NewServeMux() // Initializing the router 

	//let's add the route to ensure that the API is alive 
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request){
		w.Header().Set("content-type","application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(HealthResponse{
			Status: "OK",
			Message: "Kliops API Gatewau is running ",
		})
	})

	// let's prepare the other routes of the API, the ones protected by the Middleware 
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("POST /api/v1/upload",gatewayHandler.HandleUpload)
	apiMux.HandleFunc("GET /api/v1/price",gatewayHandler.HandlePrice)

	mux.Handle("/api/v1/", handlers.APIKeyMiddleware(apiMux)) 

	// let's configure the server 
	srv := &http.Server{
		Addr: ":8070",
		Handler: mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout: 120 * time.Second,
	}

	// now we launch the server in a new goroutine 

	go func() {
		log.Println("Starting Kliops server on port 8070 ....")
		serverListeningError := srv.ListenAndServe()
		if serverListeningError != nil && serverListeningError != http.ErrServerClosed {
			log.Fatalf("server failed to start %v",serverListeningError)
		}
	}()

	//now let's implement the graceful shutdown mechanism 
	quit := make(chan os.Signal,1)
	signal.Notify(quit,syscall.SIGINT,syscall.SIGTERM)
	<- quit 

	log.Println("shutting down gracefully ....")

	ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Second) 
	defer cancel() 

	shuttingServerDownError := srv.Shutdown(ctx)
	if shuttingServerDownError != nil {
		log.Fatalf("Server forced to shutdown %v",shuttingServerDownError)
	}

	log.Println("Server exiting ....")

}