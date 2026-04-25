package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// PersonalDetails maps to the personal_details JSONB column.
type PersonalDetails struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

// BusinessDetails maps to the business_details JSONB column.
type BusinessDetails struct {
	BusinessName          string  `json:"business_name"`
	BusinessType          string  `json:"business_type"`
	ExpectedMonthlyVolume float64 `json:"expected_monthly_volume"`
}

// Submission represents a KYC submission in the system.
type Submission struct {
	ID               uuid.UUID        `json:"submission_id" db:"id"`
	MerchantID       uuid.UUID        `json:"merchant_id" db:"merchant_id"`
	State            string           `json:"state" db:"state"`
	PersonalDetails  *PersonalDetails `json:"personal_details,omitempty"`
	BusinessDetails  *BusinessDetails `json:"business_details,omitempty"`
	ReviewerNote     *string          `json:"reviewer_note,omitempty" db:"reviewer_note"`
	CreatedAt        time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at" db:"updated_at"`

	// Computed fields (never stored in DB)
	AtRisk    bool       `json:"at_risk,omitempty" db:"-"`
	Documents []Document `json:"documents,omitempty" db:"-"`

	// Raw JSONB for DB scanning
	PersonalDetailsRaw json.RawMessage `json:"-" db:"personal_details"`
	BusinessDetailsRaw json.RawMessage `json:"-" db:"business_details"`
}

// ParseJSONFields unmarshals the raw JSONB columns into typed structs.
func (s *Submission) ParseJSONFields() error {
	if len(s.PersonalDetailsRaw) > 0 && string(s.PersonalDetailsRaw) != "null" {
		s.PersonalDetails = &PersonalDetails{}
		if err := json.Unmarshal(s.PersonalDetailsRaw, s.PersonalDetails); err != nil {
			return err
		}
	}
	if len(s.BusinessDetailsRaw) > 0 && string(s.BusinessDetailsRaw) != "null" {
		s.BusinessDetails = &BusinessDetails{}
		if err := json.Unmarshal(s.BusinessDetailsRaw, s.BusinessDetails); err != nil {
			return err
		}
	}
	return nil
}

// SubmissionQueueItem is a lightweight view used in the reviewer queue listing.
type SubmissionQueueItem struct {
	SubmissionID    uuid.UUID        `json:"submission_id"`
	MerchantID      uuid.UUID        `json:"merchant_id"`
	State           string           `json:"state"`
	AtRisk          bool             `json:"at_risk"`
	CreatedAt       time.Time        `json:"created_at"`
	PersonalDetails *PersonalDetails `json:"personal_details,omitempty"`
	BusinessDetails *BusinessDetails `json:"business_details,omitempty"`
}
