package handlers

import (
	"log/slog"
	"net/http"

	"kyc/backend/middleware"
	"kyc/backend/models"
	"kyc/backend/store"
)

// NotificationHandler handles merchant notification endpoints.
type NotificationHandler struct {
	notificationStore *store.NotificationStore
}

// NewNotificationHandler creates a new NotificationHandler.
func NewNotificationHandler(notificationStore *store.NotificationStore) *NotificationHandler {
	return &NotificationHandler{notificationStore: notificationStore}
}

// GetNotifications handles GET /api/v1/kyc/notifications.
func (h *NotificationHandler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	slog.Debug("handler.get_notifications: processing", "user_id", userID)

	notifs, err := h.notificationStore.GetNotificationsByMerchant(r.Context(), userID)
	if err != nil {
		slog.Error("handler.get_notifications: failed", "user_id", userID, "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{"failed to fetch notifications"})
		return
	}

	if notifs == nil {
		notifs = make([]models.Notification, 0)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"notifications": notifs,
	})
}
