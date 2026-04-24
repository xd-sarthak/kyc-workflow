package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalStorage implements StorageBackend using the local filesystem.
type LocalStorage struct {
	RootDir string
}

// NewLocalStorage creates a LocalStorage rooted at the given directory.
func NewLocalStorage(rootDir string) (*LocalStorage, error) {
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage root %q: %w", rootDir, err)
	}
	return &LocalStorage{RootDir: rootDir}, nil
}

func (l *LocalStorage) Save(_ context.Context, key string, r io.Reader) error {
	fullPath := filepath.Join(l.RootDir, key)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", dir, err)
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w", fullPath, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("failed to write file %q: %w", fullPath, err)
	}

	return nil
}

func (l *LocalStorage) URL(key string) string {
	return "/uploads/" + key
}

func (l *LocalStorage) Delete(_ context.Context, key string) error {
	fullPath := filepath.Join(l.RootDir, key)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file %q: %w", fullPath, err)
	}
	return nil
}
