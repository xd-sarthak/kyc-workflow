package handlers

import (
	"net/http"

	"kyc/backend/services"
)

// MetricsHandler handles metrics endpoints.
type MetricsHandler struct {
	metricsService *services.MetricsService
}

// NewMetricsHandler creates a new MetricsHandler.
func NewMetricsHandler(metricsService *services.MetricsService) *MetricsHandler {
	return &MetricsHandler{metricsService: metricsService}
}

// GetMetrics handles GET /api/v1/metrics.
func (h *MetricsHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.metricsService.GetMetrics(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{"failed to compute metrics"})
		return
	}

	writeJSON(w, http.StatusOK, metrics)
}
