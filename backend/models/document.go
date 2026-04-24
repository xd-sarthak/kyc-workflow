package models

import (
	"time"

	"github.com/google/uuid"
)

// Document represents an uploaded document associated with a KYC submission.
type Document struct {
	ID             uuid.UUID `json:"id" db:"id"`
	SubmissionID   uuid.UUID `json:"submission_id" db:"submission_id"`
	FileType       string    `json:"file_type" db:"file_type"`
	StorageKey     string    `json:"storage_key" db:"storage_key"`
	StorageBackend string    `json:"storage_backend" db:"storage_backend"`
	OriginalName   string    `json:"original_name" db:"original_name"`
	MimeType       string    `json:"mime_type" db:"mime_type"`
	SizeBytes      int64     `json:"size_bytes" db:"size_bytes"`
	UploadedAt     time.Time `json:"uploaded_at" db:"uploaded_at"`
}
