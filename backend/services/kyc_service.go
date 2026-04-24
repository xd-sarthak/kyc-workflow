package services

import (
	"context"
	"fmt"
	"log/slog"
	"mime/multipart"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"kyc/backend/models"
	"kyc/backend/storage"
	"kyc/backend/store"
)

const maxFileSize = 5 * 1024 * 1024 // 5 MB

var allowedMIMETypes = map[string]bool{
	"application/pdf": true,
	"image/jpeg":      true,
	"image/png":       true,
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// KYCService handles merchant KYC operations.
type KYCService struct {
	submissionStore  *store.SubmissionStore
	documentStore    *store.DocumentStore
	notificationStore *store.NotificationStore
	storageBackend   storage.StorageBackend
	storageType      string // "local" or "s3"
}

// NewKYCService creates a new KYCService.
func NewKYCService(
	submissionStore *store.SubmissionStore,
	documentStore *store.DocumentStore,
	notificationStore *store.NotificationStore,
	storageBackend storage.StorageBackend,
	storageType string,
) *KYCService {
	return &KYCService{
		submissionStore:   submissionStore,
		documentStore:     documentStore,
		notificationStore: notificationStore,
		storageBackend:    storageBackend,
		storageType:       storageType,
	}
}

// FileUpload represents an uploaded file from a multipart form.
type FileUpload struct {
	FileType string // "pan", "aadhaar", "bank_statement"
	Header   *multipart.FileHeader
	File     multipart.File
}

// SaveDraft creates or updates a merchant's submission in draft state.
func (s *KYCService) SaveDraft(ctx context.Context, merchantID uuid.UUID, pd *models.PersonalDetails, bd *models.BusinessDetails, files []FileUpload) (*models.Submission, error) {
	logger := slog.With("merchant_id", merchantID, "operation", "save_draft")

	// Get or create submission
	sub, err := s.submissionStore.GetSubmissionByMerchant(ctx, merchantID)
	if err != nil {
		// No existing submission — create one
		logger.Info("creating new submission")
		sub, err = s.submissionStore.CreateSubmission(ctx, merchantID)
		if err != nil {
			logger.Error("failed to create submission", "error", err)
			return nil, fmt.Errorf("failed to create submission: %w", err)
		}
		logger.Info("submission created", "submission_id", sub.ID)
	}

	// Enforce draft state
	if sub.State != string(StateDraft) {
		logger.Warn("attempted edit on non-draft submission",
			"submission_id", sub.ID,
			"current_state", sub.State,
		)
		return nil, fmt.Errorf("submission is in %q state; only drafts can be edited", sub.State)
	}

	// Update details if provided
	if pd != nil || bd != nil {
		logger.Debug("updating submission details",
			"submission_id", sub.ID,
			"has_personal", pd != nil,
			"has_business", bd != nil,
		)
		if err := s.submissionStore.UpdateSubmissionDetails(ctx, sub.ID, pd, bd); err != nil {
			logger.Error("failed to update details", "submission_id", sub.ID, "error", err)
			return nil, fmt.Errorf("failed to update submission details: %w", err)
		}
	}

	// Handle file uploads
	for _, f := range files {
		logger.Info("processing file upload",
			"submission_id", sub.ID,
			"file_type", f.FileType,
			"filename", f.Header.Filename,
			"size_bytes", f.Header.Size,
		)
		if err := s.validateAndSaveFile(ctx, sub.ID, f); err != nil {
			logger.Error("file upload failed",
				"submission_id", sub.ID,
				"file_type", f.FileType,
				"error", err,
			)
			return nil, err
		}
	}

	logger.Info("draft saved", "submission_id", sub.ID)

	// Re-fetch the submission to return current state
	return s.submissionStore.GetSubmissionByMerchant(ctx, merchantID)
}

// Submit transitions a draft submission to submitted state.
func (s *KYCService) Submit(ctx context.Context, merchantID uuid.UUID) (*models.Submission, error) {
	logger := slog.With("merchant_id", merchantID, "operation", "submit")

	sub, err := s.submissionStore.GetSubmissionByMerchant(ctx, merchantID)
	if err != nil {
		logger.Warn("no submission found for merchant")
		return nil, fmt.Errorf("no submission found for merchant")
	}

	logger = logger.With("submission_id", sub.ID, "current_state", sub.State)

	// Validate state transition
	if err := Transition(State(sub.State), StateSubmitted); err != nil {
		logger.Warn("invalid state transition", "target_state", "submitted", "error", err)
		return nil, err
	}

	// Validate completeness: personal details
	if sub.PersonalDetails == nil {
		logger.Warn("submission missing personal details")
		return nil, fmt.Errorf("personal details are required before submission")
	}
	if err := validatePersonalDetails(sub.PersonalDetails); err != nil {
		logger.Warn("personal details validation failed", "error", err)
		return nil, err
	}

	// Validate completeness: business details
	if sub.BusinessDetails == nil {
		logger.Warn("submission missing business details")
		return nil, fmt.Errorf("business details are required before submission")
	}
	if err := validateBusinessDetails(sub.BusinessDetails); err != nil {
		logger.Warn("business details validation failed", "error", err)
		return nil, err
	}

	// Validate completeness: all 3 documents
	docCount, err := s.documentStore.CountDocumentsBySubmission(ctx, sub.ID)
	if err != nil {
		logger.Error("failed to count documents", "error", err)
		return nil, fmt.Errorf("failed to check documents: %w", err)
	}
	if docCount < 3 {
		logger.Warn("insufficient documents", "count", docCount, "required", 3)
		return nil, fmt.Errorf("all three documents (pan, aadhaar, bank_statement) are required before submission")
	}

	// Perform transition
	if err := s.submissionStore.UpdateSubmissionState(ctx, sub.ID, string(StateSubmitted), nil); err != nil {
		logger.Error("failed to update state", "error", err)
		return nil, fmt.Errorf("failed to update state: %w", err)
	}

	// Create notification
	if err := s.notificationStore.CreateNotification(ctx, merchantID, string(StateSubmitted), map[string]interface{}{
		"submission_id": sub.ID.String(),
	}); err != nil {
		logger.Error("failed to create notification", "error", err)
	}

	logger.Info("submission submitted successfully",
		"from_state", "draft",
		"to_state", "submitted",
	)

	sub.State = string(StateSubmitted)
	return sub, nil
}

// GetMySubmission retrieves the authenticated merchant's own submission.
func (s *KYCService) GetMySubmission(ctx context.Context, merchantID uuid.UUID) (*models.Submission, error) {
	sub, err := s.submissionStore.GetSubmissionByMerchant(ctx, merchantID)
	if err != nil {
		slog.Debug("get_my_submission: no submission found", "merchant_id", merchantID)
		return nil, fmt.Errorf("no submission found")
	}

	docs, err := s.documentStore.GetDocumentsBySubmission(ctx, sub.ID)
	if err != nil {
		slog.Error("get_my_submission: failed to get documents",
			"submission_id", sub.ID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get documents: %w", err)
	}
	sub.Documents = docs
	sub.AtRisk = sub.State == string(StateSubmitted) && time.Since(sub.CreatedAt) > 24*time.Hour

	return sub, nil
}

// validateAndSaveFile validates file constraints and writes it to storage.
func (s *KYCService) validateAndSaveFile(ctx context.Context, submissionID uuid.UUID, f FileUpload) error {
	defer f.File.Close()

	// Validate MIME type
	contentType := f.Header.Header.Get("Content-Type")
	if !allowedMIMETypes[contentType] {
		slog.Warn("file_upload: invalid MIME type",
			"submission_id", submissionID,
			"file_type", f.FileType,
			"mime_type", contentType,
		)
		return fmt.Errorf("invalid file type %q for %s; allowed: application/pdf, image/jpeg, image/png", contentType, f.FileType)
	}

	// Validate size
	if f.Header.Size > maxFileSize {
		slog.Warn("file_upload: file too large",
			"submission_id", submissionID,
			"file_type", f.FileType,
			"size_bytes", f.Header.Size,
			"max_bytes", maxFileSize,
		)
		return fmt.Errorf("file %s exceeds maximum size of 5 MB", f.FileType)
	}

	// Generate storage key
	ext := filepath.Ext(f.Header.Filename)
	if ext == "" {
		switch contentType {
		case "application/pdf":
			ext = ".pdf"
		case "image/jpeg":
			ext = ".jpg"
		case "image/png":
			ext = ".png"
		}
	}
	storageKey := fmt.Sprintf("submissions/%s/%s/%s%s", submissionID, f.FileType, uuid.New().String(), ext)

	// Save to storage
	if err := s.storageBackend.Save(ctx, storageKey, f.File); err != nil {
		slog.Error("file_upload: storage write failed",
			"submission_id", submissionID,
			"file_type", f.FileType,
			"storage_key", storageKey,
			"error", err,
		)
		return fmt.Errorf("failed to save file %s: %w", f.FileType, err)
	}

	// Upsert document record
	doc := &models.Document{
		ID:             uuid.New(),
		SubmissionID:   submissionID,
		FileType:       f.FileType,
		StorageKey:     storageKey,
		StorageBackend: s.storageType,
		OriginalName:   f.Header.Filename,
		MimeType:       contentType,
		SizeBytes:      f.Header.Size,
	}
	if err := s.documentStore.UpsertDocument(ctx, doc); err != nil {
		slog.Error("file_upload: failed to upsert document record",
			"submission_id", submissionID,
			"file_type", f.FileType,
			"error", err,
		)
		return fmt.Errorf("failed to save document record for %s: %w", f.FileType, err)
	}

	slog.Info("file_upload: saved successfully",
		"submission_id", submissionID,
		"file_type", f.FileType,
		"storage_key", storageKey,
		"size_bytes", f.Header.Size,
	)

	return nil
}

// validatePersonalDetails checks that all required fields in personal details are present and valid.
func validatePersonalDetails(pd *models.PersonalDetails) error {
	if strings.TrimSpace(pd.FullName) == "" {
		return fmt.Errorf("full_name is required in personal details")
	}
	if strings.TrimSpace(pd.Email) == "" {
		return fmt.Errorf("email is required in personal details")
	}
	if !emailRegex.MatchString(pd.Email) {
		return fmt.Errorf("invalid email format in personal details")
	}
	if strings.TrimSpace(pd.Phone) == "" {
		return fmt.Errorf("phone is required in personal details")
	}
	return nil
}

// validateBusinessDetails checks that all required fields in business details are present and valid.
func validateBusinessDetails(bd *models.BusinessDetails) error {
	if strings.TrimSpace(bd.BusinessName) == "" {
		return fmt.Errorf("business_name is required in business details")
	}
	if strings.TrimSpace(bd.BusinessType) == "" {
		return fmt.Errorf("business_type is required in business details")
	}
	if bd.ExpectedMonthlyVolume <= 0 {
		return fmt.Errorf("expected_monthly_volume must be greater than 0")
	}
	return nil
}
