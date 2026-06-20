package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type LocalStorage struct {
	uploadDir string
}

func NewLocalStorage(uploadDir string) *LocalStorage {
	return &LocalStorage{uploadDir: uploadDir}
}

func (s *LocalStorage) Save(filename string, r io.Reader) (string, error) {
	destPath := filepath.Join(s.uploadDir, filename)
	dst, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create local file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, r); err != nil {
		return "", fmt.Errorf("failed to write local file: %w", err)
	}

	return "/uploads/" + filename, nil
}

func (s *LocalStorage) Delete(filename string) error {
	destPath := filepath.Join(s.uploadDir, filename)
	if err := os.Remove(destPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove local file: %w", err)
	}
	return nil
}
