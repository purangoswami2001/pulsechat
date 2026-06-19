package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// Maximum upload file size: 50MB
const maxUploadSize = 50 << 20

// Allowed MIME types for file uploads
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

// UploadResponse is returned after a successful file upload.
type UploadResponse struct {
	URL  string `json:"url"`
	Type string `json:"type"`
	Name string `json:"name"`
}

// UploadHandler handles file uploads. Files are stored on disk in uploadDir.
// POST /upload — multipart/form-data with field "file"
func UploadHandler(uploadDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Limit request body size
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

		// Detect MIME type from content
		buf := make([]byte, 512)
		n, _ := file.Read(buf)
		contentType := http.DetectContentType(buf[:n])

		// Seek back to start after reading header bytes
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			slog.Error("failed to seek file after content detection", "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		// Validate MIME type
		if !allowedMIMETypes[contentType] {
			respondJSONError(w, http.StatusBadRequest,
				fmt.Sprintf("File type '%s' is not allowed", contentType))
			return
		}

		// Generate unique filename preserving original extension
		ext := filepath.Ext(header.Filename)
		if ext == "" {
			// Infer extension from MIME type
			switch contentType {
			case "image/jpeg":
				ext = ".jpg"
			case "image/png":
				ext = ".png"
			case "image/gif":
				ext = ".gif"
			case "image/webp":
				ext = ".webp"
			case "image/svg+xml":
				ext = ".svg"
			case "application/pdf":
				ext = ".pdf"
			case "text/plain":
				ext = ".txt"
			case "video/mp4":
				ext = ".mp4"
			case "video/webm":
				ext = ".webm"
			case "video/ogg":
				ext = ".ogg"
			case "video/quicktime":
				ext = ".mov"
			default:
				ext = ".bin"
			}
		}

		newFilename := uuid.New().String() + strings.ToLower(ext)
		destPath := filepath.Join(uploadDir, newFilename)

		// Create destination file
		dst, err := os.Create(destPath)
		if err != nil {
			slog.Error("failed to create upload file on disk", "path", destPath, "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Failed to save file")
			return
		}
		defer dst.Close()

		// Copy file contents
		if _, err := io.Copy(dst, file); err != nil {
			slog.Error("failed to write upload file contents", "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Failed to save file")
			return
		}

		// Build URL path (relative to server root — served by static handler)
		fileURL := "/uploads/" + newFilename

		slog.Info("file uploaded successfully",
			"original_name", header.Filename,
			"stored_as", newFilename,
			"type", contentType,
			"size", header.Size,
		)

		resp := UploadResponse{
			URL:  fileURL,
			Type: contentType,
			Name: header.Filename,
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	}
}
