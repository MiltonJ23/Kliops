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
)



type HealthResponse struct {
	Status string `json:"status"`
	Message string `json:"message"`
}

func main(){
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