package handlers 

import (
	"encoding/json"
	"net/http"
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

	// let's upload the file to MiniO 
	path, uploadingFileError := h.Storage.Upload(r.Context(),"dce-entrants",header.Filename,file,header.Size,header.Header.Get("content-type"))
	if uploadingFileError != nil {
		http.Error(w,uploadingFileError.Error(),http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status" : "success",
		"path"	 : path,
	})
}

func (h *GatewayHandler) HandlePrice(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source") // ex: excel or portgres 
	code := r.URL.Query().Get("code")

	if source == "" || code == "" {
		http.Error(w,"missing required fields 'source' and 'code'",http.StatusBadRequest)
		return
	}

	price, err := h.Pricing.GetPrice(r.Context(),source,code)
	if err != nil {
		http.Error(w,err.Error(),http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code_article"	: code,
		"prix"			: price,
		"source"		: source,
	})
}