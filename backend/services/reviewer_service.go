package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"kyc/backend/models"
	"kyc/backend/store"
)

// ReviewerService handles reviewer operations.
type ReviewerService struct {
	submissionStore   *store.SubmissionStore
	documentStore     *store.DocumentStore
	notificationStore *store.NotificationStore
}

// NewReviewerService creates a new ReviewerService.
func NewReviewerService(
	submissionStore *store.SubmissionStore,
	documentStore *store.DocumentStore,
	notificationStore *store.NotificationStore,
) *ReviewerService {
	return &ReviewerService{
		submissionStore:   submissionStore,
		documentStore:     documentStore,
		notificationStore: notificationStore,
	}
}

// QueueResponse is the response for the reviewer queue endpoint.
type QueueResponse struct {
	Submissions []models.SubmissionQueueItem `json:"submissions"`
	Total       int                          `json:"total"`
}

// ListQueue returns submissions in the "submitted" state, oldest first.
func (s *ReviewerService) ListQueue(ctx context.Context, limit, offset int) (*QueueResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	slog.Debug("reviewer: listing queue",
		"limit", limit,
		"offset", offset,
	)

	subs, total, err := s.submissionStore.ListSubmissionsByState(ctx, string(StateSubmitted), limit, offset)
	if err != nil {
		slog.Error("reviewer: failed to list queue", "error", err)
		return nil, fmt.Errorf("failed to list queue: %w", err)
	}

	items := make([]models.SubmissionQueueItem, len(subs))
	atRiskCount := 0
	for i, sub := range subs {
		isAtRisk := time.Since(sub.CreatedAt) > 24*time.Hour
		if isAtRisk {
			atRiskCount++
		}
		items[i] = models.SubmissionQueueItem{
			SubmissionID:    sub.ID,
			MerchantID:     sub.MerchantID,
			State:          sub.State,
			AtRisk:         isAtRisk,
			CreatedAt:      sub.CreatedAt,
			PersonalDetails: sub.PersonalDetails,
			BusinessDetails: sub.BusinessDetails,
		}
	}

	slog.Info("reviewer: queue listed",
		"total", total,
		"returned", len(items),
		"at_risk", atRiskCount,
	)

	return &QueueResponse{
		Submissions: items,
		Total:       total,
	}, nil
}

// GetSubmissionDetail returns the full detail of a submission by ID.
func (s *ReviewerService) GetSubmissionDetail(ctx context.Context, submissionID uuid.UUID) (*models.Submission, error) {
	slog.Debug("reviewer: fetching submission detail", "submission_id", submissionID)

	sub, err := s.submissionStore.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		slog.Warn("reviewer: submission not found", "submission_id", submissionID)
		return nil, fmt.Errorf("submission not found")
	}

	docs, err := s.documentStore.GetDocumentsBySubmission(ctx, sub.ID)
	if err != nil {
		slog.Error("reviewer: failed to get documents",
			"submission_id", sub.ID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get documents: %w", err)
	}
	sub.Documents = docs
	sub.AtRisk = sub.State == string(StateSubmitted) && time.Since(sub.CreatedAt) > 24*time.Hour

	return sub, nil
}

// TransitionSubmission triggers a state transition on a submission.
func (s *ReviewerService) TransitionSubmission(ctx context.Context, submissionID uuid.UUID, toState, note string) (*models.Submission, error) {
	logger := slog.With(
		"submission_id", submissionID,
		"target_state", toState,
		"operation", "transition",
	)

	if !IsValidState(toState) {
		logger.Warn("invalid target state")
		return nil, fmt.Errorf("invalid target state: %q", toState)
	}

	sub, err := s.submissionStore.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		logger.Warn("submission not found")
		return nil, fmt.Errorf("submission not found")
	}

	logger = logger.With("from_state", sub.State, "merchant_id", sub.MerchantID)

	// Validate transition via state machine
	if err := Transition(State(sub.State), State(toState)); err != nil {
		logger.Warn("invalid state transition", "error", err)
		return nil, err
	}

	// Require note for rejections and more-info requests
	if (toState == string(StateRejected) || toState == string(StateMoreInfoRequested)) && note == "" {
		logger.Warn("note required but not provided")
		return nil, fmt.Errorf("note is required when transitioning to %q", toState)
	}

	// Update state
	var notePtr *string
	if note != "" {
		notePtr = &note
	}
	if err := s.submissionStore.UpdateSubmissionState(ctx, sub.ID, toState, notePtr); err != nil {
		logger.Error("failed to update state", "error", err)
		return nil, fmt.Errorf("failed to update state: %w", err)
	}

	// Create notification
	payload := map[string]interface{}{
		"submission_id": sub.ID.String(),
	}
	if note != "" {
		payload["reviewer_note"] = note
	}
	if err := s.notificationStore.CreateNotification(ctx, sub.MerchantID, toState, payload); err != nil {
		logger.Error("failed to create notification", "error", err)
	}

	logger.Info("state transition completed",
		"from_state", sub.State,
		"to_state", toState,
		"has_note", note != "",
	)

	sub.State = toState
	if notePtr != nil {
		sub.ReviewerNote = notePtr
	}
	return sub, nil
}
