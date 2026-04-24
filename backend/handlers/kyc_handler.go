package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"kyc/backend/middleware"
	"kyc/backend/models"
	"kyc/backend/services"
)

// KYCHandler handles merchant KYC endpoints.
type KYCHandler struct {
	kycService *services.KYCService
}

// NewKYCHandler creates a new KYCHandler.
func NewKYCHandler(kycService *services.KYCService) *KYCHandler {
	return &KYCHandler{kycService: kycService}
}

// SaveDraft handles POST /api/v1/kyc/save-draft (multipart/form-data).
func (h *KYCHandler) SaveDraft(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	// Parse multipart form (32 MB max memory)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		slog.Warn("handler.save_draft: invalid multipart form",
			"user_id", userID,
			"error", err,
		)
		writeJSON(w, http.StatusBadRequest, errorResponse{"invalid multipart form"})
		return
	}

	// Parse personal details if present
	var pd *models.PersonalDetails
	if pdStr := r.FormValue("personal_details"); pdStr != "" {
		pd = &models.PersonalDetails{}
		if err := json.Unmarshal([]byte(pdStr), pd); err != nil {
			slog.Warn("handler.save_draft: invalid personal_details JSON",
				"user_id", userID,
				"error", err,
			)
			writeJSON(w, http.StatusBadRequest, errorResponse{"invalid personal_details JSON"})
			return
		}
	}

	// Parse business details if present
	var bd *models.BusinessDetails
	if bdStr := r.FormValue("business_details"); bdStr != "" {
		bd = &models.BusinessDetails{}
		if err := json.Unmarshal([]byte(bdStr), bd); err != nil {
			slog.Warn("handler.save_draft: invalid business_details JSON",
				"user_id", userID,
				"error", err,
			)
			writeJSON(w, http.StatusBadRequest, errorResponse{"invalid business_details JSON"})
			return
		}
	}

	// Collect file uploads
	var files []services.FileUpload
	for _, fileType := range []string{"pan", "aadhaar", "bank_statement"} {
		file, header, err := r.FormFile(fileType)
		if err != nil {
			continue // Field not present — skip (partial update)
		}
		files = append(files, services.FileUpload{
			FileType: fileType,
			Header:   header,
			File:     file,
		})
	}

	slog.Debug("handler.save_draft: processing",
		"user_id", userID,
		"has_personal", pd != nil,
		"has_business", bd != nil,
		"file_count", len(files),
	)

	sub, err := h.kycService.SaveDraft(r.Context(), userID, pd, bd, files)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"submission_id": sub.ID,
		"state":         sub.State,
	})
}

// Submit handles POST /api/v1/kyc/submit.
func (h *KYCHandler) Submit(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	slog.Debug("handler.submit: processing", "user_id", userID)

	sub, err := h.kycService.Submit(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"submission_id": sub.ID,
		"state":         sub.State,
	})
}

// GetMySubmission handles GET /api/v1/kyc/me.
func (h *KYCHandler) GetMySubmission(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	slog.Debug("handler.get_my_submission: processing", "user_id", userID)

	sub, err := h.kycService.GetMySubmission(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, sub)
}
