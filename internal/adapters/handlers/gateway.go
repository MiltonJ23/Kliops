package handlers 

import (
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/MiltonJ23/Kliops/internal/core/ports"
	"github.com/MiltonJ23/Kliops/internal/core/services"
)


type GatewayHandler struct {
	Storage ports.FileStorage 
	Pricing *services.PricingService
}

func NewGatewayHandler(storage ports.FileStorage,pricing *services.PricingService) *GatewayHandler {
	return &GatewayHandler{
		Storage: storage,
		Pricing: pricing,
	}
}

// HandleUpload manage the reception of the DCE 
func (h *GatewayHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	// let's set the limit to 50 MB 
	parsingFileSizeError := r.ParseMultipartForm(50 << 20)
	if parsingFileSizeError != nil {
		http.Error(w,"file too heavy or invalid request",http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("document")
	if err != nil {
		http.Error(w,"missing field 'document' ",http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Sanitize filename to prevent path traversal
	filename := filepath.Base(header.Filename)
	if strings.Contains(filename, "..") || len(filename) > 255 || strings.ContainsAny(filename, "/\\") {
		http.Error(w, "invalid filename", http.StatusBadRequest)
		return
	}

	// let's upload the file to MiniO 
	path, uploadingFileError := h.Storage.Upload(r.Context(),"dce-entrants",filename,file,header.Size,header.Header.Get("content-type"))
	if uploadingFileError != nil {
		log.Printf("Error uploading file: %v", uploadingFileError)
		http.Error(w,"failed to upload file",http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status" : "success",
		"path"	 : path,
	}); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

func (h *GatewayHandler) HandlePrice(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source") // ex: excel or postgres 
	code := r.URL.Query().Get("code")

	if source == "" || code == "" {
		http.Error(w,"missing required fields 'source' and 'code'",http.StatusBadRequest)
		return
	}

	price, err := h.Pricing.GetPrice(r.Context(),source,code)
	if err != nil {
		// Log the full error server-side
		log.Printf("Error getting price for code %s from source %s: %v", code, source, err)
		// Return generic message to client
		http.Error(w,"internal server error",http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"code_article"	: code,
		"prix"			: price,
		"source"		: source,
	}); err != nil {
		log.Printf("Error encoding price response: %v", err)
	}
}