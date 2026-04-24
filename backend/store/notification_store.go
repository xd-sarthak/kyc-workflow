package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"kyc/backend/models"
)

// NotificationStore handles database operations for notifications.
type NotificationStore struct {
	db *DB
}

// NewNotificationStore creates a new NotificationStore.
func NewNotificationStore(db *DB) *NotificationStore {
	return &NotificationStore{db: db}
}

// CreateNotification inserts a new notification.
func (s *NotificationStore) CreateNotification(ctx context.Context, merchantID uuid.UUID, eventType string, payload map[string]interface{}) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal notification payload: %w", err)
	}

	_, err = s.db.Pool.Exec(ctx,
		`INSERT INTO notifications (merchant_id, event_type, payload) VALUES ($1, $2, $3)`,
		merchantID, eventType, payloadJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}
	return nil
}

// GetNotificationsByMerchant retrieves all notifications for a merchant, newest first.
func (s *NotificationStore) GetNotificationsByMerchant(ctx context.Context, merchantID uuid.UUID) ([]models.Notification, error) {
	rows, err := s.db.Pool.Query(ctx,
		`SELECT id, merchant_id, event_type, payload, created_at
		 FROM notifications WHERE merchant_id = $1 ORDER BY created_at DESC`,
		merchantID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get notifications: %w", err)
	}
	defer rows.Close()

	var notifs []models.Notification
	for rows.Next() {
		var n models.Notification
		var payloadJSON []byte
		if err := rows.Scan(&n.ID, &n.MerchantID, &n.EventType, &payloadJSON, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		if err := json.Unmarshal(payloadJSON, &n.Payload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal notification payload: %w", err)
		}
		notifs = append(notifs, n)
	}
	return notifs, nil
}
