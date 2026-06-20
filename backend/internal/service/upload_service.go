package service

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/pulsechat/backend/internal/storage"
)

type UploadService struct {
	store storage.Storage
}

func NewUploadService(store storage.Storage) *UploadService {
	return &UploadService{store: store}
}

func (s *UploadService) SaveFile(filename string, contentType string, r io.Reader) (string, error) {
	ext := filepath.Ext(filename)
	if ext == "" {
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
	url, err := s.store.Save(newFilename, r)
	if err != nil {
		return "", fmt.Errorf("failed to upload: %w", err)
	}

	return url, nil
}

func (s *UploadService) DeleteFile(url string) error {
	filename := filepath.Base(url)
	return s.store.Delete(filename)
}
