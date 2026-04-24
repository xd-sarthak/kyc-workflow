package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"kyc/backend/services"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	authService *services.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type signupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

// Signup handles POST /api/v1/signup.
func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("handler.signup: invalid request body", "error", err)
		writeJSON(w, http.StatusBadRequest, errorResponse{"invalid request body"})
		return
	}

	slog.Debug("handler.signup: processing", "email", req.Email, "role", req.Role)

	token, err := h.authService.Signup(r.Context(), req.Email, req.Password, req.Role)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, tokenResponse{Token: token})
}

// Login handles POST /api/v1/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("handler.login: invalid request body", "error", err)
		writeJSON(w, http.StatusBadRequest, errorResponse{"invalid request body"})
		return
	}

	slog.Debug("handler.login: processing", "email", req.Email)

	token, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, tokenResponse{Token: token})
}

// --- Helpers ---

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
