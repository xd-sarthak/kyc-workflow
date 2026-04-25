package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"kyc/backend/models"
)

// SubmissionStore handles database operations for KYC submissions.
type SubmissionStore struct {
	db *DB
}

// NewSubmissionStore creates a new SubmissionStore.
func NewSubmissionStore(db *DB) *SubmissionStore {
	return &SubmissionStore{db: db}
}

// CreateSubmission inserts a new submission in draft state.
func (s *SubmissionStore) CreateSubmission(ctx context.Context, merchantID uuid.UUID) (*models.Submission, error) {
	sub := &models.Submission{}
	err := s.db.Pool.QueryRow(ctx,
		`INSERT INTO kyc_submissions (merchant_id, state) VALUES ($1, 'draft')
		 RETURNING id, merchant_id, state, personal_details, business_details, reviewer_note, created_at, updated_at`,
		merchantID,
	).Scan(&sub.ID, &sub.MerchantID, &sub.State, &sub.PersonalDetailsRaw, &sub.BusinessDetailsRaw,
		&sub.ReviewerNote, &sub.CreatedAt, &sub.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create submission: %w", err)
	}
	_ = sub.ParseJSONFields()
	return sub, nil
}

// GetSubmissionByMerchant retrieves the submission for a given merchant.
func (s *SubmissionStore) GetSubmissionByMerchant(ctx context.Context, merchantID uuid.UUID) (*models.Submission, error) {
	sub := &models.Submission{}
	err := s.db.Pool.QueryRow(ctx,
		`SELECT id, merchant_id, state, personal_details, business_details, reviewer_note, created_at, updated_at
		 FROM kyc_submissions WHERE merchant_id = $1`,
		merchantID,
	).Scan(&sub.ID, &sub.MerchantID, &sub.State, &sub.PersonalDetailsRaw, &sub.BusinessDetailsRaw,
		&sub.ReviewerNote, &sub.CreatedAt, &sub.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("submission not found: %w", err)
	}
	if err := sub.ParseJSONFields(); err != nil {
		return nil, fmt.Errorf("failed to parse submission JSON fields: %w", err)
	}
	return sub, nil
}

// GetSubmissionByID retrieves a submission by its ID.
func (s *SubmissionStore) GetSubmissionByID(ctx context.Context, id uuid.UUID) (*models.Submission, error) {
	sub := &models.Submission{}
	err := s.db.Pool.QueryRow(ctx,
		`SELECT id, merchant_id, state, personal_details, business_details, reviewer_note, created_at, updated_at
		 FROM kyc_submissions WHERE id = $1`,
		id,
	).Scan(&sub.ID, &sub.MerchantID, &sub.State, &sub.PersonalDetailsRaw, &sub.BusinessDetailsRaw,
		&sub.ReviewerNote, &sub.CreatedAt, &sub.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("submission not found: %w", err)
	}
	if err := sub.ParseJSONFields(); err != nil {
		return nil, fmt.Errorf("failed to parse submission JSON fields: %w", err)
	}
	return sub, nil
}

// UpdateSubmissionDetails updates the personal/business details and optionally the state.
func (s *SubmissionStore) UpdateSubmissionDetails(ctx context.Context, id uuid.UUID, personalDetails *models.PersonalDetails, businessDetails *models.BusinessDetails) error {
	var pdJSON, bdJSON []byte
	var err error

	if personalDetails != nil {
		pdJSON, err = json.Marshal(personalDetails)
		if err != nil {
			return fmt.Errorf("failed to marshal personal details: %w", err)
		}
	}
	if businessDetails != nil {
		bdJSON, err = json.Marshal(businessDetails)
		if err != nil {
			return fmt.Errorf("failed to marshal business details: %w", err)
		}
	}

	// Build dynamic update — only update non-nil fields
	query := `UPDATE kyc_submissions SET updated_at = now()`
	args := []interface{}{}
	argIdx := 1

	if personalDetails != nil {
		query += fmt.Sprintf(", personal_details = $%d", argIdx)
		args = append(args, pdJSON)
		argIdx++
	}
	if businessDetails != nil {
		query += fmt.Sprintf(", business_details = $%d", argIdx)
		args = append(args, bdJSON)
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argIdx)
	args = append(args, id)

	_, err = s.db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update submission: %w", err)
	}
	return nil
}

// UpdateSubmissionState updates the state and optionally the reviewer note.
func (s *SubmissionStore) UpdateSubmissionState(ctx context.Context, id uuid.UUID, state string, reviewerNote *string) error {
	var err error
	if reviewerNote != nil {
		_, err = s.db.Pool.Exec(ctx,
			`UPDATE kyc_submissions SET state = $1, reviewer_note = $2, updated_at = now() WHERE id = $3`,
			state, *reviewerNote, id)
	} else {
		_, err = s.db.Pool.Exec(ctx,
			`UPDATE kyc_submissions SET state = $1, updated_at = now() WHERE id = $2`,
			state, id)
	}
	if err != nil {
		return fmt.Errorf("failed to update submission state: %w", err)
	}
	return nil
}

// ListSubmissionsByState retrieves submissions in a given state, oldest first.
func (s *SubmissionStore) ListSubmissionsByState(ctx context.Context, state string, limit, offset int) ([]models.Submission, int, error) {
	// Get total count
	var total int
	err := s.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM kyc_submissions WHERE state = $1`, state,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count submissions: %w", err)
	}

	rows, err := s.db.Pool.Query(ctx,
		`SELECT id, merchant_id, state, personal_details, business_details, reviewer_note, created_at, updated_at
		 FROM kyc_submissions WHERE state = $1 ORDER BY created_at ASC LIMIT $2 OFFSET $3`,
		state, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list submissions: %w", err)
	}
	defer rows.Close()

	var subs []models.Submission
	for rows.Next() {
		var sub models.Submission
		if err := rows.Scan(&sub.ID, &sub.MerchantID, &sub.State, &sub.PersonalDetailsRaw, &sub.BusinessDetailsRaw,
			&sub.ReviewerNote, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan submission: %w", err)
		}
		_ = sub.ParseJSONFields()
		subs = append(subs, sub)
	}

	return subs, total, nil
}

// CountSubmissionsByState returns the count of submissions in a given state.
func (s *SubmissionStore) CountSubmissionsByState(ctx context.Context, state string) (int, error) {
	var count int
	err := s.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM kyc_submissions WHERE state = $1`, state,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count submissions: %w", err)
	}
	return count, nil
}

// GetAverageTimeInQueue returns the average seconds submissions have been in the submitted state.
func (s *SubmissionStore) GetAverageTimeInQueue(ctx context.Context) (float64, error) {
	var avg *float64
	err := s.db.Pool.QueryRow(ctx,
		`SELECT AVG(EXTRACT(EPOCH FROM (now() - created_at))) FROM kyc_submissions WHERE state = 'submitted'`,
	).Scan(&avg)
	if err != nil {
		return 0, fmt.Errorf("failed to get avg time in queue: %w", err)
	}
	if avg == nil {
		return 0, nil
	}
	return *avg, nil
}

// GetApprovalRateLast7Days returns the approval rate for submissions finalized in the last 7 days.
func (s *SubmissionStore) GetApprovalRateLast7Days(ctx context.Context) (float64, error) {
	cutoff := time.Now().Add(-7 * 24 * time.Hour)

	var approved, total int
	err := s.db.Pool.QueryRow(ctx,
		`SELECT
			COALESCE(SUM(CASE WHEN state = 'approved' THEN 1 ELSE 0 END), 0),
			COALESCE(COUNT(*), 0)
		 FROM kyc_submissions
		 WHERE state IN ('approved', 'rejected') AND updated_at >= $1`,
		cutoff,
	).Scan(&approved, &total)
	if err != nil {
		return 0, fmt.Errorf("failed to get approval rate: %w", err)
	}
	if total == 0 {
		return 0, nil
	}
	return float64(approved) / float64(total), nil
}
