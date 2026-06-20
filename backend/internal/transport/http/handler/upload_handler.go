package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/pulsechat/backend/internal/service"
)

const maxUploadSize = 50 << 20

var allowedMIMETypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/gif":       true,
	"image/webp":      true,
	"image/svg+xml":   true,
	"application/pdf": true,
	"text/plain":      true,
	"application/zip": true,
	"video/mp4":       true,
	"video/webm":      true,
	"video/ogg":       true,
	"video/quicktime": true,
}

type UploadHandler struct {
	uploadService *service.UploadService
}

func NewUploadHandler(uploadService *service.UploadService) *UploadHandler {
	return &UploadHandler{uploadService: uploadService}
}

type UploadResponse struct {
	URL  string `json:"url"`
	Type string `json:"type"`
	Name string `json:"name"`
}

func (h *UploadHandler) Upload() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			slog.Warn("file upload too large or invalid multipart", "error", err)
			respondJSONError(w, http.StatusBadRequest, "File too large. Maximum 50MB allowed.")
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			slog.Warn("missing file field in upload", "error", err)
			respondJSONError(w, http.StatusBadRequest, "File field 'file' is required")
			return
		}
		defer file.Close()

		buf := make([]byte, 512)
		n, _ := file.Read(buf)
		contentType := http.DetectContentType(buf[:n])

		if _, err := file.Seek(0, io.SeekStart); err != nil {
			slog.Error("failed to seek file after content detection", "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		if !allowedMIMETypes[contentType] {
			respondJSONError(w, http.StatusBadRequest,
				fmt.Sprintf("File type '%s' is not allowed", contentType))
			return
		}

		url, err := h.uploadService.SaveFile(header.Filename, contentType, file)
		if err != nil {
			slog.Error("failed to save file", "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Failed to save file")
			return
		}

		resp := UploadResponse{
			URL:  url,
			Type: contentType,
			Name: header.Filename,
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	}
}
