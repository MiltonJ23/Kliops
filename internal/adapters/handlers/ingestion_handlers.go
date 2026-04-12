package handlers 

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/MiltonJ23/Kliops/internal/core/ports"
	"github.com/MiltonJ23/Kliops/internal/core/services"
)

type IngestionHandler struct {
	ArchiveService *services.ArchiveService
	FileStorage    ports.FileStorage
}

func NewIngestionHandler(archive *services.ArchiveService, storage ports.FileStorage) *IngestionHandler {
	return &IngestionHandler{
		ArchiveService: archive,
		FileStorage:    storage,
	}
}

// POST /api/v1/ingest/archive
// Reçoit le ZIP contenant manifest.csv et les PDF
func (h *IngestionHandler) UploadArchiveZip(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form avec une limite de mémoire (ex: 32 MB en RAM, le reste sur disque)
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("archive")
	if err != nil {
		http.Error(w, "archive field is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if filepath.Ext(header.Filename) != ".zip" {
		http.Error(w, "File must be a ZIP archive", http.StatusBadRequest)
		return
	}

	// Appel du service de traitement d'archive (défini précédemment)
	err = h.ArchiveService.ProcessZipArchive(r.Context(), file, header.Size)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to process archive: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"message": "Archive received, processing started asynchronously"}`))
}

// POST /api/v1/ingest/mercuriale
// Reçoit le fichier Excel des prix
func (h *IngestionHandler) UploadMercuriale(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("excel_file")
	if err != nil {
		http.Error(w, "excel_file field is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if filepath.Ext(header.Filename) != ".xlsx" {
		http.Error(w, "File must be a valid XLSX", http.StatusBadRequest)
		return
	}

	// Sauvegarde sur MinIO dans un bucket dédié "kliops-config"
	// L'application utilisera ce chemin quand on interrogera la stratégie Excel
	path, err := h.FileStorage.Upload(r.Context(), "kliops-config", "mercuriale_current.xlsx", file, header.Size, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	if err != nil {
		http.Error(w, "Failed to upload to MinIO", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"message": "Mercuriale updated successfully", "path": "%s"}`, path)))
}

// POST /api/v1/ingest/template
// Reçoit le fichier DOCX de charte d'entreprise
func (h *IngestionHandler) UploadTemplateDocx(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("template_file")
	if err != nil {
		http.Error(w, "template_file field is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if filepath.Ext(header.Filename) != ".docx" {
		http.Error(w, "File must be a valid DOCX", http.StatusBadRequest)
		return
	}

	// Sauvegarde sur MinIO 
	path, err := h.FileStorage.Upload(r.Context(), "kliops-config", "template_charte.docx", file, header.Size, "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	if err != nil {
		http.Error(w, "Failed to upload to MinIO", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"message": "Template DOCX uploaded successfully", "path": "%s"}`, path)))
}