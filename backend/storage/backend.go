package storage

import (
	"context"
	"io"
)

// StorageBackend abstracts file storage operations.
// Implementations must be safe for concurrent use.
type StorageBackend interface {
	// Save writes the content from r to the given key.
	Save(ctx context.Context, key string, r io.Reader) error

	// URL returns a URL or path to access the stored file.
	URL(key string) string

	// Delete removes the file at the given key.
	Delete(ctx context.Context, key string) error
}
