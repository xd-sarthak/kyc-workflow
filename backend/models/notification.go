package models

import (
	"time"

	"github.com/google/uuid"
)

// Notification records a state-change event for a merchant's submission.
type Notification struct {
	ID         uuid.UUID              `json:"id" db:"id"`
	MerchantID uuid.UUID             `json:"merchant_id" db:"merchant_id"`
	EventType  string                 `json:"event_type" db:"event_type"`
	Payload    map[string]interface{} `json:"payload" db:"payload"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
}
