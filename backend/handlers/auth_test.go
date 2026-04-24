package handlers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"kyc/backend/testutil"
)

func TestAuth_MerchantCanAccessOwnSubmission(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	merchantAID, jwtA := testutil.CreateMerchant(t, env, "merchant_a@test.com")
	testutil.CreateSubmission(t, env, merchantAID, "draft")

	_, jwtB := testutil.CreateMerchant(t, env, "merchant_b@test.com")

	// merchant_a sees own submission
	rr := testutil.MakeRequest(t, env, "GET", "/api/v1/kyc/me", nil, jwtA)
	testutil.AssertStatus(t, rr, 200)

	var resp map[string]interface{}
	testutil.DecodeResponse(t, rr, &resp)
	assert.Equal(t, merchantAID.String(), resp["merchant_id"])

	// merchant_b sees nothing (no submission)
	rr2 := testutil.MakeRequest(t, env, "GET", "/api/v1/kyc/me", nil, jwtB)
	testutil.AssertStatus(t, rr2, 404)
}

func TestAuth_MerchantCannotAccessReviewerRoutes(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	_, jwtA := testutil.CreateMerchant(t, env, "merchant@test.com")

	// merchant → reviewer queue
	rr := testutil.MakeRequest(t, env, "GET", "/api/v1/reviewer/queue", nil, jwtA)
	testutil.AssertStatus(t, rr, 403)
}

func TestAuth_MerchantCannotAccessOtherSubmission(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	merchantAID, _ := testutil.CreateMerchant(t, env, "merchant_a@test.com")
	subID := testutil.CreateSubmission(t, env, merchantAID, "submitted")

	_, jwtB := testutil.CreateMerchant(t, env, "merchant_b@test.com")

	// merchant_b trying to access merchant_a's submission via reviewer endpoint
	rr := testutil.MakeRequest(t, env, "GET", "/api/v1/reviewer/"+subID.String(), nil, jwtB)
	testutil.AssertStatus(t, rr, 403)
}

func TestAuth_ReviewerCanAccessQueue(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	merchantID, _ := testutil.CreateMerchant(t, env, "merchant@test.com")
	subID := testutil.CreateSubmission(t, env, merchantID, "submitted")
	_, jwtReviewer := testutil.CreateReviewer(t, env, "reviewer@test.com")

	// reviewer → queue
	rr := testutil.MakeRequest(t, env, "GET", "/api/v1/reviewer/queue", nil, jwtReviewer)
	testutil.AssertStatus(t, rr, 200)

	// reviewer → specific submission
	rr2 := testutil.MakeRequest(t, env, "GET", "/api/v1/reviewer/"+subID.String(), nil, jwtReviewer)
	testutil.AssertStatus(t, rr2, 200)
}

func TestAuth_UnauthenticatedRequest(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	rr := testutil.MakeRequest(t, env, "GET", "/api/v1/kyc/me", nil, "")
	testutil.AssertStatus(t, rr, 401)
}

func TestAuth_ExpiredToken(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	// Manually craft an expired JWT
	expiredToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjEwMDAwMDAwMDAsInJvbGUiOiJtZXJjaGFudCIsInN1YiI6IjAwMDAwMDAwLTAwMDAtMDAwMC0wMDAwLTAwMDAwMDAwMDAwMCJ9.invalid"

	rr := testutil.MakeRequest(t, env, "GET", "/api/v1/kyc/me", nil, expiredToken)
	testutil.AssertStatus(t, rr, 401)
}

func TestAuth_TamperedToken(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	_, jwt := testutil.CreateMerchant(t, env, "merchant@test.com")

	// Tamper by appending garbage
	tampered := jwt + "tampered"

	rr := testutil.MakeRequest(t, env, "GET", "/api/v1/kyc/me", nil, tampered)
	testutil.AssertStatus(t, rr, 401)
}

func TestAuth_MerchantResponseNeverLeaksOtherData(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	merchantAID, jwtA := testutil.CreateMerchant(t, env, "merchant_a@test.com")
	testutil.CreateSubmission(t, env, merchantAID, "submitted")

	merchantBID, _ := testutil.CreateMerchant(t, env, "merchant_b@test.com")
	testutil.CreateSubmission(t, env, merchantBID, "submitted")

	// merchant_a can only see own data
	rr := testutil.MakeRequest(t, env, "GET", "/api/v1/kyc/me", nil, jwtA)
	testutil.AssertStatus(t, rr, 200)

	body := rr.Body.String()
	assert.NotContains(t, body, merchantBID.String(), "response must not contain other merchant's ID")
}
