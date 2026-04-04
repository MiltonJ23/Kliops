package handlers

import (
	"net/http"
	"os"
)

// APIKeyMiddleware will block the request that don't have the proper header 
func APIKeyMiddleware( next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		expectedApiKey := os.Getenv("API_KEY_SECRET")
		providedKey := r.Header.Get("X-API-KEY")

		if expectedApiKey == "" {
			http.Error(w,"Server Configuration Error",http.StatusInternalServerError)
			return
		}

		if providedKey != expectedApiKey {
			http.Error(w,"Unauthorized : Invalid API Key", http.StatusUnauthorized)
			return 
		}
		next.ServeHTTP(w,r)
	})
}

