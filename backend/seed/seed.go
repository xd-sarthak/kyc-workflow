package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// Run seeds the database with initial data if it is empty.
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

	slog.Info("seed: seeding database with comprehensive test data")

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), 12)
	if err != nil {
		return fmt.Errorf("failed to hash seed password: %w", err)
	}
	hp := string(hashedPassword)

	// Helper for executing inserts
	execQuery := func(q string, args ...any) error {
		_, err := pool.Exec(ctx, q, args...)
		return err
	}

	// 1. Create reviewer account
	reviewerID := uuid.New()
	if err := execQuery(`INSERT INTO users (id, email, password, role) VALUES ($1, $2, $3, $4)`, reviewerID, "reviewer@kyc.dev", hp, "reviewer"); err != nil {
		return err
	}
	slog.Info("seed: created reviewer", "email", "reviewer@kyc.dev")

	// SEED SCENARIOS:
	type Scenario struct {
		Email          string
		State          string
		BusinessName   string
		CreatedHours   int
		UpdatedHours   int
		ReviewerNote   string
		HasDocs        bool
		Notifications  []string // event types
	}

	scenarios := []Scenario{
		{
			Email:        "merchant.draft@kyc.dev",
			State:        "draft",
			BusinessName: "Draft Agency",
			CreatedHours: 48,
			UpdatedHours: 48,
			HasDocs:      false,
		},
		{
			Email:        "merchant.submitted@kyc.dev",
			State:        "submitted",
			BusinessName: "Fresh E-commerce",
			CreatedHours: 2,
			UpdatedHours: 2,
			HasDocs:      true,
			Notifications: []string{"submitted"},
		},
		{
			Email:        "merchant.atrisk@kyc.dev",
			State:        "submitted",
			BusinessName: "Old Freelancer",
			CreatedHours: 48, // 48 hours ago -> marks it as at_risk
			UpdatedHours: 48,
			HasDocs:      true,
			Notifications: []string{"submitted"},
		},
		{
			Email:        "merchant.moreinfo@kyc.dev",
			State:        "more_info_requested",
			BusinessName: "InfoNeeded Corp",
			CreatedHours: 72,
			UpdatedHours: 5,
			ReviewerNote: "The bank statement is blurry. Please upload a clear PDF of the last 3 months.",
			HasDocs:      true,
			Notifications: []string{"submitted", "under_review", "more_info_requested"},
		},
		{
			Email:        "merchant.approved@kyc.dev",
			State:        "approved",
			BusinessName: "Approved Tech",
			CreatedHours: 120,
			UpdatedHours: 24,
			HasDocs:      true,
			Notifications: []string{"submitted", "under_review", "approved"},
		},
	}

	for _, s := range scenarios {
		mID := uuid.New()
		if err := execQuery(`INSERT INTO users (id, email, password, role) VALUES ($1, $2, $3, $4)`, mID, s.Email, hp, "merchant"); err != nil {
			return err
		}

		pd, _ := json.Marshal(map[string]interface{}{
			"full_name": s.BusinessName + " Owner",
			"email":     s.Email,
			"phone":     "+91 98765 00000",
		})
		
		var bd []byte
		if s.State != "draft" {
			bType := "Agency"
			if s.BusinessName == "Fresh E-commerce" { bType = "E-commerce" }
			if s.BusinessName == "Old Freelancer" { bType = "Freelancer" }
			if s.BusinessName == "InfoNeeded Corp" { bType = "Other" }
			
			bd, _ = json.Marshal(map[string]interface{}{
				"business_name":           s.BusinessName,
				"business_type":           bType,
				"expected_monthly_volume": 25000.0,
			})
		} else {
			bd, _ = json.Marshal(map[string]interface{}{})
		}

		now := time.Now()
		cTime := now.Add(-time.Hour * time.Duration(s.CreatedHours))
		uTime := now.Add(-time.Hour * time.Duration(s.UpdatedHours))

		subID := uuid.New()
		
		var rNote *string
		if s.ReviewerNote != "" {
			rNote = &s.ReviewerNote
		}

		if err := execQuery(
			`INSERT INTO kyc_submissions (id, merchant_id, state, personal_details, business_details, reviewer_note, created_at, updated_at) 
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			subID, mID, s.State, pd, bd, rNote, cTime, uTime,
		); err != nil {
			return err
		}

		// Documents
		if s.HasDocs {
			for _, fileType := range []string{"pan", "aadhaar", "bank_statement"} {
				if err := execQuery(
					`INSERT INTO documents (submission_id, file_type, storage_key, storage_backend, original_name, mime_type, size_bytes, uploaded_at)
					 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
					subID, fileType,
					fmt.Sprintf("submissions/%s/%s/seed-placeholder.pdf", subID, fileType),
					"local", fileType+"_seed.pdf", "application/pdf", 1024, cTime,
				); err != nil {
					return err
				}
			}
		}

		// Notifications (timeline events)
		for i, nEvent := range s.Notifications {
			payload := map[string]interface{}{}
			if nEvent == "more_info_requested" || nEvent == "rejected" {
				payload["note"] = s.ReviewerNote
			}
			pBytes, _ := json.Marshal(payload)
			
			// Compute fake timestamp for the event based on order
			nTime := cTime.Add(time.Hour * time.Duration(i*2))
			if nTime.After(uTime) { nTime = uTime }

			if err := execQuery(
				`INSERT INTO notifications (merchant_id, event_type, payload, created_at) VALUES ($1, $2, $3, $4)`,
				mID, nEvent, pBytes, nTime,
			); err != nil {
				return err
			}
		}

		slog.Info("seed: created scenario", "email", s.Email, "state", s.State)
	}

	slog.Info("seed: database seeding complete")
	return nil
}
