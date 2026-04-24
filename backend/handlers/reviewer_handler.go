package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"kyc/backend/services"
)

// ReviewerHandler handles reviewer endpoints.
type ReviewerHandler struct {
	reviewerService *services.ReviewerService
}

// NewReviewerHandler creates a new ReviewerHandler.
func NewReviewerHandler(reviewerService *services.ReviewerService) *ReviewerHandler {
	return &ReviewerHandler{reviewerService: reviewerService}
}

// ListQueue handles GET /api/v1/reviewer/queue.
func (h *ReviewerHandler) ListQueue(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = 20
	}

	slog.Debug("handler.list_queue: processing",
		"limit", limit,
		"offset", offset,
	)

	resp, err := h.reviewerService.ListQueue(r.Context(), limit, offset)
	if err != nil {
		slog.Error("handler.list_queue: failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{"failed to fetch queue"})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetSubmission handles GET /api/v1/reviewer/{id}.
func (h *ReviewerHandler) GetSubmission(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		slog.Warn("handler.get_submission: invalid ID", "id", idStr)
		writeJSON(w, http.StatusBadRequest, errorResponse{"invalid submission ID"})
		return
	}

	slog.Debug("handler.get_submission: processing", "submission_id", id)

	sub, err := h.reviewerService.GetSubmissionDetail(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, sub)
}

type transitionRequest struct {
	To   string `json:"to"`
	Note string `json:"note"`
}

// TransitionSubmission handles POST /api/v1/reviewer/{id}/transition.
func (h *ReviewerHandler) TransitionSubmission(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		slog.Warn("handler.transition: invalid ID", "id", idStr)
		writeJSON(w, http.StatusBadRequest, errorResponse{"invalid submission ID"})
		return
	}

	var req transitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("handler.transition: invalid request body",
			"submission_id", id,
			"error", err,
		)
		writeJSON(w, http.StatusBadRequest, errorResponse{"invalid request body"})
		return
	}

	slog.Debug("handler.transition: processing",
		"submission_id", id,
		"target_state", req.To,
		"has_note", req.Note != "",
	)

	sub, err := h.reviewerService.TransitionSubmission(r.Context(), id, req.To, req.Note)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"submission_id": sub.ID,
		"state":         sub.State,
	})
}
