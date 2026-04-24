package services

import (
	"context"
	"fmt"

	"kyc/backend/store"
)

// MetricsService handles metrics computation.
type MetricsService struct {
	submissionStore *store.SubmissionStore
}

// NewMetricsService creates a new MetricsService.
func NewMetricsService(submissionStore *store.SubmissionStore) *MetricsService {
	return &MetricsService{submissionStore: submissionStore}
}

// MetricsResponse holds the computed metrics.
type MetricsResponse struct {
	QueueSize              int     `json:"queue_size"`
	AvgTimeInQueueSeconds  float64 `json:"avg_time_in_queue_seconds"`
	ApprovalRateLast7Days  float64 `json:"approval_rate_last_7d"`
}

// GetMetrics computes all metrics at query time.
func (s *MetricsService) GetMetrics(ctx context.Context) (*MetricsResponse, error) {
	queueSize, err := s.submissionStore.CountSubmissionsByState(ctx, string(StateSubmitted))
	if err != nil {
		return nil, fmt.Errorf("failed to get queue size: %w", err)
	}

	avgTime, err := s.submissionStore.GetAverageTimeInQueue(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get avg time in queue: %w", err)
	}

	approvalRate, err := s.submissionStore.GetApprovalRateLast7Days(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get approval rate: %w", err)
	}

	return &MetricsResponse{
		QueueSize:             queueSize,
		AvgTimeInQueueSeconds: avgTime,
		ApprovalRateLast7Days: approvalRate,
	}, nil
}
