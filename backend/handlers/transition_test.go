package handlers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"kyc/backend/testutil"
)

// --- Legal transitions ---

func TestTransition_DraftToSubmitted(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	merchantID, jwt := testutil.CreateMerchant(t, env, "m@test.com")
	testutil.CreateSubmission(t, env, merchantID, "draft")

	rr := testutil.MakeRequest(t, env, "POST", "/api/v1/kyc/submit", map[string]string{}, jwt)
	testutil.AssertStatus(t, rr, 200)

	var resp map[string]interface{}
	testutil.DecodeResponse(t, rr, &resp)
	assert.Equal(t, "submitted", resp["state"])
}

func TestTransition_SubmittedToUnderReview(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	merchantID, _ := testutil.CreateMerchant(t, env, "m@test.com")
	subID := testutil.CreateSubmission(t, env, merchantID, "submitted")
	_, reviewerJWT := testutil.CreateReviewer(t, env, "r@test.com")

	rr := testutil.MakeRequest(t, env, "POST", "/api/v1/reviewer/"+subID.String()+"/transition",
		map[string]string{"to": "under_review"}, reviewerJWT)
	testutil.AssertStatus(t, rr, 200)

	var resp map[string]interface{}
	testutil.DecodeResponse(t, rr, &resp)
	assert.Equal(t, "under_review", resp["state"])
}

func TestTransition_UnderReviewToApproved(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	merchantID, _ := testutil.CreateMerchant(t, env, "m@test.com")
	subID := testutil.CreateSubmission(t, env, merchantID, "under_review")
	_, reviewerJWT := testutil.CreateReviewer(t, env, "r@test.com")

	rr := testutil.MakeRequest(t, env, "POST", "/api/v1/reviewer/"+subID.String()+"/transition",
		map[string]string{"to": "approved"}, reviewerJWT)
	testutil.AssertStatus(t, rr, 200)

	var resp map[string]interface{}
	testutil.DecodeResponse(t, rr, &resp)
	assert.Equal(t, "approved", resp["state"])
}

func TestTransition_UnderReviewToRejected(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	merchantID, _ := testutil.CreateMerchant(t, env, "m@test.com")
	subID := testutil.CreateSubmission(t, env, merchantID, "under_review")
	_, reviewerJWT := testutil.CreateReviewer(t, env, "r@test.com")

	rr := testutil.MakeRequest(t, env, "POST", "/api/v1/reviewer/"+subID.String()+"/transition",
		map[string]interface{}{"to": "rejected", "note": "Incomplete documents"}, reviewerJWT)
	testutil.AssertStatus(t, rr, 200)

	var resp map[string]interface{}
	testutil.DecodeResponse(t, rr, &resp)
	assert.Equal(t, "rejected", resp["state"])
}

func TestTransition_UnderReviewToMoreInfoRequested(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	merchantID, _ := testutil.CreateMerchant(t, env, "m@test.com")
	subID := testutil.CreateSubmission(t, env, merchantID, "under_review")
	_, reviewerJWT := testutil.CreateReviewer(t, env, "r@test.com")

	rr := testutil.MakeRequest(t, env, "POST", "/api/v1/reviewer/"+subID.String()+"/transition",
		map[string]interface{}{"to": "more_info_requested", "note": "Need bank statement"}, reviewerJWT)
	testutil.AssertStatus(t, rr, 200)

	var resp map[string]interface{}
	testutil.DecodeResponse(t, rr, &resp)
	assert.Equal(t, "more_info_requested", resp["state"])
}

func TestTransition_MoreInfoToResubmit(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	merchantID, merchantJWT := testutil.CreateMerchant(t, env, "m@test.com")
	testutil.CreateSubmission(t, env, merchantID, "more_info_requested")

	rr := testutil.MakeRequest(t, env, "POST", "/api/v1/kyc/submit", map[string]string{}, merchantJWT)
	testutil.AssertStatus(t, rr, 200)

	var resp map[string]interface{}
	testutil.DecodeResponse(t, rr, &resp)
	assert.Equal(t, "submitted", resp["state"])
}

// --- Illegal transitions ---

func TestTransition_ApprovedToDraft_Illegal(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	merchantID, _ := testutil.CreateMerchant(t, env, "m@test.com")
	subID := testutil.CreateSubmission(t, env, merchantID, "approved")
	_, reviewerJWT := testutil.CreateReviewer(t, env, "r@test.com")

	rr := testutil.MakeRequest(t, env, "POST", "/api/v1/reviewer/"+subID.String()+"/transition",
		map[string]string{"to": "draft"}, reviewerJWT)
	testutil.AssertErrorContains(t, rr, 400, "approved")
}

func TestTransition_ApprovedToSubmitted_Illegal(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	merchantID, _ := testutil.CreateMerchant(t, env, "m@test.com")
	subID := testutil.CreateSubmission(t, env, merchantID, "approved")
	_, reviewerJWT := testutil.CreateReviewer(t, env, "r@test.com")

	rr := testutil.MakeRequest(t, env, "POST", "/api/v1/reviewer/"+subID.String()+"/transition",
		map[string]string{"to": "submitted"}, reviewerJWT)
	testutil.AssertErrorContains(t, rr, 400, "approved")
}

func TestTransition_RejectedToApproved_Illegal(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	merchantID, _ := testutil.CreateMerchant(t, env, "m@test.com")
	subID := testutil.CreateSubmission(t, env, merchantID, "rejected")
	_, reviewerJWT := testutil.CreateReviewer(t, env, "r@test.com")

	rr := testutil.MakeRequest(t, env, "POST", "/api/v1/reviewer/"+subID.String()+"/transition",
		map[string]string{"to": "approved"}, reviewerJWT)
	testutil.AssertErrorContains(t, rr, 400, "rejected")
}

func TestTransition_SubmittedToApproved_SkipReview(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	merchantID, _ := testutil.CreateMerchant(t, env, "m@test.com")
	subID := testutil.CreateSubmission(t, env, merchantID, "submitted")
	_, reviewerJWT := testutil.CreateReviewer(t, env, "r@test.com")

	rr := testutil.MakeRequest(t, env, "POST", "/api/v1/reviewer/"+subID.String()+"/transition",
		map[string]string{"to": "approved"}, reviewerJWT)
	testutil.AssertErrorContains(t, rr, 400, "submitted")
}

func TestTransition_UnderReviewToDraft_Illegal(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	merchantID, _ := testutil.CreateMerchant(t, env, "m@test.com")
	subID := testutil.CreateSubmission(t, env, merchantID, "under_review")
	_, reviewerJWT := testutil.CreateReviewer(t, env, "r@test.com")

	rr := testutil.MakeRequest(t, env, "POST", "/api/v1/reviewer/"+subID.String()+"/transition",
		map[string]string{"to": "draft"}, reviewerJWT)
	testutil.AssertErrorContains(t, rr, 400, "under_review")
}

func TestTransition_ApprovedToApproved_Illegal(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	merchantID, _ := testutil.CreateMerchant(t, env, "m@test.com")
	subID := testutil.CreateSubmission(t, env, merchantID, "approved")
	_, reviewerJWT := testutil.CreateReviewer(t, env, "r@test.com")

	rr := testutil.MakeRequest(t, env, "POST", "/api/v1/reviewer/"+subID.String()+"/transition",
		map[string]string{"to": "approved"}, reviewerJWT)
	testutil.AssertErrorContains(t, rr, 400, "approved")
}

// --- Missing note ---

func TestTransition_RejectWithoutNote(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	merchantID, _ := testutil.CreateMerchant(t, env, "m@test.com")
	subID := testutil.CreateSubmission(t, env, merchantID, "under_review")
	_, reviewerJWT := testutil.CreateReviewer(t, env, "r@test.com")

	rr := testutil.MakeRequest(t, env, "POST", "/api/v1/reviewer/"+subID.String()+"/transition",
		map[string]string{"to": "rejected"}, reviewerJWT)
	testutil.AssertErrorContains(t, rr, 400, "note")
}

func TestTransition_MoreInfoWithoutNote(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	merchantID, _ := testutil.CreateMerchant(t, env, "m@test.com")
	subID := testutil.CreateSubmission(t, env, merchantID, "under_review")
	_, reviewerJWT := testutil.CreateReviewer(t, env, "r@test.com")

	rr := testutil.MakeRequest(t, env, "POST", "/api/v1/reviewer/"+subID.String()+"/transition",
		map[string]string{"to": "more_info_requested"}, reviewerJWT)
	testutil.AssertErrorContains(t, rr, 400, "note")
}

// --- Role enforcement on transitions ---

func TestTransition_MerchantCannotCallReviewerTransition(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	merchantID, merchantJWT := testutil.CreateMerchant(t, env, "m@test.com")
	subID := testutil.CreateSubmission(t, env, merchantID, "submitted")

	rr := testutil.MakeRequest(t, env, "POST", "/api/v1/reviewer/"+subID.String()+"/transition",
		map[string]string{"to": "under_review"}, merchantJWT)
	testutil.AssertStatus(t, rr, 403)
}

func TestTransition_ReviewerCannotCallMerchantSubmit(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	_, reviewerJWT := testutil.CreateReviewer(t, env, "r@test.com")

	rr := testutil.MakeRequest(t, env, "POST", "/api/v1/kyc/submit", map[string]string{}, reviewerJWT)
	testutil.AssertStatus(t, rr, 403)
}
