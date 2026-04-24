package handlers_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kyc/backend/testutil"
)

func TestNotification_CreatedOnSubmit(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	merchantID, jwt := testutil.CreateMerchant(t, env, "m@test.com")
	testutil.CreateSubmission(t, env, merchantID, "draft")

	// Submit triggers draft → submitted, which creates a notification
	rr := testutil.MakeRequest(t, env, "POST", "/api/v1/kyc/submit", map[string]string{}, jwt)
	testutil.AssertStatus(t, rr, 200)

	// Check notification row exists
	count := testutil.CountRows(t, env.Pool, "notifications",
		"merchant_id = $1 AND event_type = 'submitted'", merchantID)
	assert.Equal(t, 1, count)
}

func TestNotification_ReviewerTransitionCreatesNotification(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	merchantID, _ := testutil.CreateMerchant(t, env, "m@test.com")
	subID := testutil.CreateSubmission(t, env, merchantID, "submitted")
	_, reviewerJWT := testutil.CreateReviewer(t, env, "r@test.com")

	// Clear any notifications from CreateSubmission transitions
	env.Pool.Exec(context.Background(), "TRUNCATE notifications CASCADE")

	// Transition submitted → under_review
	rr := testutil.MakeRequest(t, env, "POST", "/api/v1/reviewer/"+subID.String()+"/transition",
		map[string]string{"to": "under_review"}, reviewerJWT)
	testutil.AssertStatus(t, rr, 200)

	count := testutil.CountRows(t, env.Pool, "notifications",
		"merchant_id = $1 AND event_type = 'under_review'", merchantID)
	assert.Equal(t, 1, count)
}

func TestNotification_TwoTransitionsCreatesTwoRows(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	merchantID, _ := testutil.CreateMerchant(t, env, "m@test.com")
	subID := testutil.CreateSubmission(t, env, merchantID, "submitted")
	_, reviewerJWT := testutil.CreateReviewer(t, env, "r@test.com")

	// Clear any previous notifications
	env.Pool.Exec(context.Background(), "TRUNCATE notifications CASCADE")

	// Transition 1: submitted → under_review
	rr := testutil.MakeRequest(t, env, "POST", "/api/v1/reviewer/"+subID.String()+"/transition",
		map[string]string{"to": "under_review"}, reviewerJWT)
	testutil.AssertStatus(t, rr, 200)

	// Transition 2: under_review → approved
	rr2 := testutil.MakeRequest(t, env, "POST", "/api/v1/reviewer/"+subID.String()+"/transition",
		map[string]string{"to": "approved"}, reviewerJWT)
	testutil.AssertStatus(t, rr2, 200)

	count := testutil.CountRows(t, env.Pool, "notifications", "merchant_id = $1", merchantID)
	assert.Equal(t, 2, count)
}

func TestNotification_FailedTransitionNoRow(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	merchantID, _ := testutil.CreateMerchant(t, env, "m@test.com")
	subID := testutil.CreateSubmission(t, env, merchantID, "approved")
	_, reviewerJWT := testutil.CreateReviewer(t, env, "r@test.com")

	// Clear any previous notifications
	env.Pool.Exec(context.Background(), "TRUNCATE notifications CASCADE")

	// Attempt illegal transition: approved → draft
	rr := testutil.MakeRequest(t, env, "POST", "/api/v1/reviewer/"+subID.String()+"/transition",
		map[string]string{"to": "draft"}, reviewerJWT)
	testutil.AssertStatus(t, rr, 400)

	// Zero notifications created
	count := testutil.CountRows(t, env.Pool, "notifications", "merchant_id = $1", merchantID)
	assert.Equal(t, 0, count)
}

func TestNotification_PayloadContainsSubmissionID(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	merchantID, _ := testutil.CreateMerchant(t, env, "m@test.com")
	subID := testutil.CreateSubmission(t, env, merchantID, "submitted")
	_, reviewerJWT := testutil.CreateReviewer(t, env, "r@test.com")

	env.Pool.Exec(context.Background(), "TRUNCATE notifications CASCADE")

	rr := testutil.MakeRequest(t, env, "POST", "/api/v1/reviewer/"+subID.String()+"/transition",
		map[string]string{"to": "under_review"}, reviewerJWT)
	testutil.AssertStatus(t, rr, 200)

	var payload []byte
	err := env.Pool.QueryRow(context.Background(),
		`SELECT payload FROM notifications WHERE merchant_id = $1 ORDER BY created_at DESC LIMIT 1`,
		merchantID).Scan(&payload)
	require.NoError(t, err)
	assert.Contains(t, string(payload), subID.String())
}

func TestNotification_RejectionPayloadContainsNote(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	merchantID, _ := testutil.CreateMerchant(t, env, "m@test.com")
	subID := testutil.CreateSubmission(t, env, merchantID, "under_review")
	_, reviewerJWT := testutil.CreateReviewer(t, env, "r@test.com")

	env.Pool.Exec(context.Background(), "TRUNCATE notifications CASCADE")

	rr := testutil.MakeRequest(t, env, "POST", "/api/v1/reviewer/"+subID.String()+"/transition",
		map[string]interface{}{"to": "rejected", "note": "Documents are blurry"}, reviewerJWT)
	testutil.AssertStatus(t, rr, 200)

	var payload []byte
	err := env.Pool.QueryRow(context.Background(),
		`SELECT payload FROM notifications WHERE merchant_id = $1 AND event_type = 'rejected'`,
		merchantID).Scan(&payload)
	require.NoError(t, err)
	assert.Contains(t, string(payload), "Documents are blurry")
}
