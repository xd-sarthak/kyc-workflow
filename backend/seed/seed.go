package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// Run seeds the database with initial data if it is empty.
// Idempotent — checks before inserting.
func Run(ctx context.Context, pool *pgxpool.Pool) error {
	slog.Info("seed: checking if seed data is needed")

	var userCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}
	if userCount > 0 {
		slog.Info("seed: data already exists, skipping", "user_count", userCount)
		return nil
	}

	slog.Info("seed: seeding database with demo data")

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), 12)
	if err != nil {
		return fmt.Errorf("failed to hash seed password: %w", err)
	}
	hp := string(hashedPassword)

	// 1. Create reviewer account
	reviewerID := uuid.New()
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, email, password, role) VALUES ($1, $2, $3, $4)`,
		reviewerID, "reviewer@kyc.dev", hp, "reviewer",
	)
	if err != nil {
		return fmt.Errorf("failed to create reviewer: %w", err)
	}
	slog.Info("seed: created user",
		"email", "reviewer@kyc.dev",
		"role", "reviewer",
		"user_id", reviewerID,
	)

	// 2. Create merchant with draft submission
	merchantDraftID := uuid.New()
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, email, password, role) VALUES ($1, $2, $3, $4)`,
		merchantDraftID, "merchant.draft@kyc.dev", hp, "merchant",
	)
	if err != nil {
		return fmt.Errorf("failed to create draft merchant: %w", err)
	}

	partialPD, _ := json.Marshal(map[string]string{
		"full_name": "Draft Merchant",
		"email":     "merchant.draft@kyc.dev",
	})
	draftSubID := uuid.New()
	_, err = pool.Exec(ctx,
		`INSERT INTO kyc_submissions (id, merchant_id, state, personal_details) VALUES ($1, $2, 'draft', $3)`,
		draftSubID, merchantDraftID, partialPD,
	)
	if err != nil {
		return fmt.Errorf("failed to create draft submission: %w", err)
	}
	slog.Info("seed: created user with draft submission",
		"email", "merchant.draft@kyc.dev",
		"role", "merchant",
		"user_id", merchantDraftID,
		"submission_id", draftSubID,
		"state", "draft",
	)

	// 3. Create merchant with under_review submission
	merchantReviewID := uuid.New()
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, email, password, role) VALUES ($1, $2, $3, $4)`,
		merchantReviewID, "merchant.review@kyc.dev", hp, "merchant",
	)
	if err != nil {
		return fmt.Errorf("failed to create review merchant: %w", err)
	}

	fullPD, _ := json.Marshal(map[string]interface{}{
		"full_name": "Review Merchant",
		"email":     "merchant.review@kyc.dev",
		"phone":     "+91-9876543210",
	})
	fullBD, _ := json.Marshal(map[string]interface{}{
		"business_name":           "Review Corp",
		"business_type":           "retail",
		"expected_monthly_volume": 50000.0,
	})

	submissionID := uuid.New()
	_, err = pool.Exec(ctx,
		`INSERT INTO kyc_submissions (id, merchant_id, state, personal_details, business_details)
		 VALUES ($1, $2, 'under_review', $3, $4)`,
		submissionID, merchantReviewID, fullPD, fullBD,
	)
	if err != nil {
		return fmt.Errorf("failed to create review submission: %w", err)
	}

	// Create placeholder document records for all 3 types
	for _, fileType := range []string{"pan", "aadhaar", "bank_statement"} {
		_, err = pool.Exec(ctx,
			`INSERT INTO documents (submission_id, file_type, storage_key, storage_backend, original_name, mime_type, size_bytes)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			submissionID, fileType,
			fmt.Sprintf("submissions/%s/%s/seed-placeholder.pdf", submissionID, fileType),
			"local",
			fileType+"_seed.pdf",
			"application/pdf",
			1024,
		)
		if err != nil {
			return fmt.Errorf("failed to create seed document %s: %w", fileType, err)
		}
	}
	slog.Info("seed: created user with under_review submission",
		"email", "merchant.review@kyc.dev",
		"role", "merchant",
		"user_id", merchantReviewID,
		"submission_id", submissionID,
		"state", "under_review",
		"documents", 3,
	)

	slog.Info("seed: database seeding complete",
		"users_created", 3,
		"submissions_created", 2,
	)
	return nil
}
