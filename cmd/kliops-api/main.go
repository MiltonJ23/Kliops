package main 

import (
	"encoding/json"
	"net/http"
	"time"
	"log"
	"os/signal"
	"syscall"
	"context"
	"os"
	"github.com/joho/godotenv"
	"github.com/MiltonJ23/Kliops/internal/adapters/repositories"
	"github.com/MiltonJ23/Kliops/internal/adapters/handlers"
	"github.com/MiltonJ23/Kliops/internal/core/services"
	"github.com/xuri/excelize/v2"
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



	minioStorage, err := repositories.NewMinioStorage("localhost:9000",os.Getenv("MINIO_ROOT_USER"),os.Getenv("MINIO_ROOT_PASSWORD"),false)
	if err != nil {
		log.Fatalf("unable to reach miniO : %v",err)
	}

	pricingService := services.NewPricingService() 
	if _, err := os.Stat("dummy_prices.xlsx"); os.IsNotExist(err) {
		log.Println("Création du fichier Excel de test...")
		f := excelize.NewFile()
		// excelize crée une feuille "Sheet1" par défaut, on la renomme en "Prix" comme attendu par l'adapter
		f.SetSheetName("Sheet1", "Prix")
		f.SetCellValue("Prix", "A1", "ART01")
		f.SetCellValue("Prix", "B1", 150.50) // Prix du ART01
		f.SetCellValue("Prix", "A2", "ART02")
		f.SetCellValue("Prix", "B2", 45.00)
		
		if err := f.SaveAs("dummy_prices.xlsx"); err != nil {
			log.Fatalf("Impossible de sauvegarder l'Excel de test : %v", err)
		}
		f.Close()
	}

	excelStrategy := repositories.NewExcelPricing("dummy_prices.xlsx")
	erpStrategy := repositories.NewERPPricing("http://api.erp-btp.local") 

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