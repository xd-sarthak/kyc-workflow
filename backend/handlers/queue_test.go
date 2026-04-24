package handlers_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kyc/backend/testutil"
)

func TestQueue_Empty(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	_, jwt := testutil.CreateReviewer(t, env, "r@test.com")

	rr := testutil.MakeRequest(t, env, "GET", "/api/v1/reviewer/queue", nil, jwt)
	testutil.AssertStatus(t, rr, 200)

	var resp map[string]interface{}
	testutil.DecodeResponse(t, rr, &resp)
	assert.Equal(t, float64(0), resp["total"])
	subs := resp["submissions"].([]interface{})
	assert.Len(t, subs, 0)
}

func TestQueue_OldestFirst(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	_, jwt := testutil.CreateReviewer(t, env, "r@test.com")

	// Create 3 merchants with submissions at different times
	m1, _ := testutil.CreateMerchant(t, env, "m1@test.com")
	m2, _ := testutil.CreateMerchant(t, env, "m2@test.com")
	m3, _ := testutil.CreateMerchant(t, env, "m3@test.com")

	sub3 := testutil.CreateSubmissionWithTime(t, env, m3, "submitted", time.Now().Add(-1*time.Hour))
	sub1 := testutil.CreateSubmissionWithTime(t, env, m1, "submitted", time.Now().Add(-3*time.Hour))
	sub2 := testutil.CreateSubmissionWithTime(t, env, m2, "submitted", time.Now().Add(-2*time.Hour))

	rr := testutil.MakeRequest(t, env, "GET", "/api/v1/reviewer/queue", nil, jwt)
	testutil.AssertStatus(t, rr, 200)

	var resp map[string]interface{}
	testutil.DecodeResponse(t, rr, &resp)
	subs := resp["submissions"].([]interface{})
	require.Len(t, subs, 3)

	// Assert order: oldest first (sub1 → sub2 → sub3)
	assert.Equal(t, sub1.String(), subs[0].(map[string]interface{})["submission_id"])
	assert.Equal(t, sub2.String(), subs[1].(map[string]interface{})["submission_id"])
	assert.Equal(t, sub3.String(), subs[2].(map[string]interface{})["submission_id"])
}

func TestQueue_AtRisk_25Hours(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	_, jwt := testutil.CreateReviewer(t, env, "r@test.com")
	m, _ := testutil.CreateMerchant(t, env, "m@test.com")

	testutil.CreateSubmissionWithTime(t, env, m, "submitted", time.Now().Add(-25*time.Hour))

	rr := testutil.MakeRequest(t, env, "GET", "/api/v1/reviewer/queue", nil, jwt)
	testutil.AssertStatus(t, rr, 200)

	var resp map[string]interface{}
	testutil.DecodeResponse(t, rr, &resp)
	subs := resp["submissions"].([]interface{})
	require.Len(t, subs, 1)
	assert.Equal(t, true, subs[0].(map[string]interface{})["at_risk"])
}

func TestQueue_NotAtRisk_23Hours(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	_, jwt := testutil.CreateReviewer(t, env, "r@test.com")
	m, _ := testutil.CreateMerchant(t, env, "m@test.com")

	testutil.CreateSubmissionWithTime(t, env, m, "submitted", time.Now().Add(-23*time.Hour))

	rr := testutil.MakeRequest(t, env, "GET", "/api/v1/reviewer/queue", nil, jwt)
	testutil.AssertStatus(t, rr, 200)

	var resp map[string]interface{}
	testutil.DecodeResponse(t, rr, &resp)
	subs := resp["submissions"].([]interface{})
	require.Len(t, subs, 1)
	assert.Equal(t, false, subs[0].(map[string]interface{})["at_risk"])
}

func TestQueue_Pagination(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	_, jwt := testutil.CreateReviewer(t, env, "r@test.com")

	// Create 5 submitted submissions
	for i := 0; i < 5; i++ {
		m, _ := testutil.CreateMerchant(t, env, "m"+string(rune('a'+i))+"@test.com")
		testutil.CreateSubmission(t, env, m, "submitted")
	}

	// Page 1: limit=2, offset=0
	rr := testutil.MakeRequest(t, env, "GET", "/api/v1/reviewer/queue?limit=2&offset=0", nil, jwt)
	testutil.AssertStatus(t, rr, 200)

	var resp1 map[string]interface{}
	testutil.DecodeResponse(t, rr, &resp1)
	assert.Equal(t, float64(5), resp1["total"])
	assert.Len(t, resp1["submissions"].([]interface{}), 2)

	// Page 3: limit=2, offset=4 → 1 result
	rr2 := testutil.MakeRequest(t, env, "GET", "/api/v1/reviewer/queue?limit=2&offset=4", nil, jwt)
	testutil.AssertStatus(t, rr2, 200)

	var resp2 map[string]interface{}
	testutil.DecodeResponse(t, rr2, &resp2)
	assert.Equal(t, float64(5), resp2["total"])
	assert.Len(t, resp2["submissions"].([]interface{}), 1)
}

func TestQueue_ExcludesNonSubmittedStates(t *testing.T) {
	env := testutil.SetupTestDB(t)
	testutil.CleanDB(t, env.Pool)

	_, jwt := testutil.CreateReviewer(t, env, "r@test.com")

	m1, _ := testutil.CreateMerchant(t, env, "m1@test.com")
	m2, _ := testutil.CreateMerchant(t, env, "m2@test.com")
	m3, _ := testutil.CreateMerchant(t, env, "m3@test.com")

	testutil.CreateSubmission(t, env, m1, "draft")
	testutil.CreateSubmission(t, env, m2, "approved")
	testutil.CreateSubmission(t, env, m3, "submitted")

	rr := testutil.MakeRequest(t, env, "GET", "/api/v1/reviewer/queue", nil, jwt)
	testutil.AssertStatus(t, rr, 200)

	var resp map[string]interface{}
	testutil.DecodeResponse(t, rr, &resp)
	assert.Equal(t, float64(1), resp["total"])
	subs := resp["submissions"].([]interface{})
	assert.Len(t, subs, 1)

	// Verify it's the submitted one
	sub0 := subs[0].(map[string]interface{})
	assert.Equal(t, "submitted", sub0["state"])

	// Ensure draft/approved merchant IDs are not in response
	body, _ := json.Marshal(resp)
	assert.NotContains(t, string(body), m1.String())
	assert.NotContains(t, string(body), m2.String())
}
