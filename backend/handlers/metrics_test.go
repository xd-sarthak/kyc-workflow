package handlers_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"kyc/backend/testutil"
)

func TestMetrics_EmptyDB(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	_, jwt := testutil.CreateReviewer(t, env, "r@test.com")

	rr := testutil.MakeRequest(t, env, "GET", "/api/v1/metrics/", nil, jwt)
	testutil.AssertStatus(t, rr, 200)

	var resp map[string]interface{}
	testutil.DecodeResponse(t, rr, &resp)
	assert.Equal(t, float64(0), resp["queue_size"])
	assert.Equal(t, float64(0), resp["avg_time_in_queue_seconds"])
	assert.Equal(t, float64(0), resp["approval_rate_last_7d"])
}

func TestMetrics_QueueSize(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	_, jwt := testutil.CreateReviewer(t, env, "r@test.com")

	for i := 0; i < 3; i++ {
		m, _ := testutil.CreateMerchant(t, env, "m"+string(rune('a'+i))+"@test.com")
		testutil.CreateSubmission(t, env, m, "submitted")
	}

	rr := testutil.MakeRequest(t, env, "GET", "/api/v1/metrics/", nil, jwt)
	testutil.AssertStatus(t, rr, 200)

	var resp map[string]interface{}
	testutil.DecodeResponse(t, rr, &resp)
	assert.Equal(t, float64(3), resp["queue_size"])
}

func TestMetrics_ApprovalRate(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	_, jwt := testutil.CreateReviewer(t, env, "r@test.com")

	// 2 approved + 1 rejected in last 7 days → 0.67
	m1, _ := testutil.CreateMerchant(t, env, "m1@test.com")
	m2, _ := testutil.CreateMerchant(t, env, "m2@test.com")
	m3, _ := testutil.CreateMerchant(t, env, "m3@test.com")

	testutil.CreateSubmission(t, env, m1, "approved")
	testutil.CreateSubmission(t, env, m2, "approved")
	testutil.CreateSubmission(t, env, m3, "rejected")

	rr := testutil.MakeRequest(t, env, "GET", "/api/v1/metrics/", nil, jwt)
	testutil.AssertStatus(t, rr, 200)

	var resp map[string]interface{}
	testutil.DecodeResponse(t, rr, &resp)
	rate := resp["approval_rate_last_7d"].(float64)
	assert.InDelta(t, 0.67, rate, 0.01)
}

func TestMetrics_ApprovalRateExcludesOldData(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	_, jwt := testutil.CreateReviewer(t, env, "r@test.com")

	// 1 approved 8 days ago — should NOT count
	m1, _ := testutil.CreateMerchant(t, env, "m1@test.com")
	subID1 := testutil.CreateSubmission(t, env, m1, "approved")
	// Set updated_at to 8 days ago
	env.Pool.Exec(t.Context(),
		`UPDATE kyc_submissions SET updated_at = $1 WHERE id = $2`,
		time.Now().Add(-8*24*time.Hour), subID1)

	// 1 approved yesterday — should count
	m2, _ := testutil.CreateMerchant(t, env, "m2@test.com")
	testutil.CreateSubmission(t, env, m2, "approved")

	rr := testutil.MakeRequest(t, env, "GET", "/api/v1/metrics/", nil, jwt)
	testutil.AssertStatus(t, rr, 200)

	var resp map[string]interface{}
	testutil.DecodeResponse(t, rr, &resp)
	assert.Equal(t, float64(1), resp["approval_rate_last_7d"])
}

func TestMetrics_AvgTimeInQueue(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	_, jwt := testutil.CreateReviewer(t, env, "r@test.com")

	// One submission 2 hours old, one 4 hours old → avg ~3 hours = ~10800s
	m1, _ := testutil.CreateMerchant(t, env, "m1@test.com")
	m2, _ := testutil.CreateMerchant(t, env, "m2@test.com")

	testutil.CreateSubmissionWithTime(t, env, m1, "submitted", time.Now().Add(-2*time.Hour))
	testutil.CreateSubmissionWithTime(t, env, m2, "submitted", time.Now().Add(-4*time.Hour))

	rr := testutil.MakeRequest(t, env, "GET", "/api/v1/metrics/", nil, jwt)
	testutil.AssertStatus(t, rr, 200)

	var resp map[string]interface{}
	testutil.DecodeResponse(t, rr, &resp)
	avg := resp["avg_time_in_queue_seconds"].(float64)
	assert.True(t, math.Abs(avg-10800) < 120, "expected avg ~10800s, got %f", avg)
}
