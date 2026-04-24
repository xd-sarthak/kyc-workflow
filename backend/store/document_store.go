package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"kyc/backend/models"
)

// DocumentStore handles database operations for documents.
type DocumentStore struct {
	db *DB
}

// NewDocumentStore creates a new DocumentStore.
func NewDocumentStore(db *DB) *DocumentStore {
	return &DocumentStore{db: db}
}

// UpsertDocument inserts or updates a document for a given submission and file type.
// Uses the unique constraint on (submission_id, file_type) to handle upserts.
func (s *DocumentStore) UpsertDocument(ctx context.Context, doc *models.Document) error {
	_, err := s.db.Pool.Exec(ctx,
		`INSERT INTO documents (id, submission_id, file_type, storage_key, storage_backend, original_name, mime_type, size_bytes)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (submission_id, file_type)
		 DO UPDATE SET storage_key = $4, storage_backend = $5, original_name = $6, mime_type = $7, size_bytes = $8, uploaded_at = now()`,
		doc.ID, doc.SubmissionID, doc.FileType, doc.StorageKey, doc.StorageBackend,
		doc.OriginalName, doc.MimeType, doc.SizeBytes,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert document: %w", err)
	}
	return nil
}

// GetDocumentsBySubmission retrieves all documents for a given submission.
func (s *DocumentStore) GetDocumentsBySubmission(ctx context.Context, submissionID uuid.UUID) ([]models.Document, error) {
	rows, err := s.db.Pool.Query(ctx,
		`SELECT id, submission_id, file_type, storage_key, storage_backend, original_name, mime_type, size_bytes, uploaded_at
		 FROM documents WHERE submission_id = $1 ORDER BY file_type`,
		submissionID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get documents: %w", err)
	}
	defer rows.Close()

	var docs []models.Document
	for rows.Next() {
		var doc models.Document
		if err := rows.Scan(&doc.ID, &doc.SubmissionID, &doc.FileType, &doc.StorageKey, &doc.StorageBackend,
			&doc.OriginalName, &doc.MimeType, &doc.SizeBytes, &doc.UploadedAt); err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

// CountDocumentsBySubmission returns how many documents exist for a submission.
func (s *DocumentStore) CountDocumentsBySubmission(ctx context.Context, submissionID uuid.UUID) (int, error) {
	var count int
	err := s.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM documents WHERE submission_id = $1`, submissionID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}
	return count, nil
}
