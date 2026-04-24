# EXPLAINER.md

## 1. The State Machine

The state machine lives in a single file: `backend/services/statemachine.go`. There's one map and one function. That's it.

```go
var allowedTransitions = map[State][]State{
    StateDraft:             {StateSubmitted},
    StateSubmitted:         {StateUnderReview},
    StateUnderReview:       {StateApproved, StateRejected, StateMoreInfoRequested},
    StateMoreInfoRequested: {StateSubmitted},
}

func Transition(from, to State) error {
    allowed, ok := allowedTransitions[from]
    if !ok {
        return fmt.Errorf("invalid transition: no transitions defined from state %q", from)
    }
    for _, s := range allowed {
        if s == to {
            return nil
        }
    }
    return fmt.Errorf("invalid transition: %q → %q is not allowed", from, to)
}
```

How it prevents illegal transitions: the `allowedTransitions` map is the single source of truth. If a transition isn't in the map, it doesn't happen. Period. Terminal states like `approved` and `rejected` have no entry in the map at all, so calling `Transition("approved", "draft")` hits the `!ok` branch and returns an error immediately. There's no way to bypass this because every service method that changes state calls `Transition()` first. The handlers just pass the error back as HTTP 400.

I deliberately avoided scattering `if/else` or `switch` checks across handlers. Every state change in the system, whether it's a merchant submitting or a reviewer approving, flows through this one function. If I need to add a new transition later, I add one line to the map. Nothing else changes.

The tests in `services/statemachine_test.go` cover all 6 legal transitions and 10 illegal ones, including edge cases like skipping review (`submitted` -> `approved`), going backwards from terminal states, and passing in garbage state strings.

---

## 2. The Upload

File validation happens in `backend/services/kyc_service.go` inside `validateAndSaveFile`. Two checks run before anything touches storage:

```go
const maxFileSize = 5 * 1024 * 1024 // 5 MB

var allowedMIMETypes = map[string]bool{
    "application/pdf": true,
    "image/jpeg":      true,
    "image/png":       true,
}

func (s *KYCService) validateAndSaveFile(ctx context.Context, submissionID uuid.UUID, f FileUpload) error {
    defer f.File.Close()

    // Validate MIME type
    contentType := f.Header.Header.Get("Content-Type")
    if !allowedMIMETypes[contentType] {
        return fmt.Errorf("invalid file type %q for %s; allowed: application/pdf, image/jpeg, image/png", contentType, f.FileType)
    }

    // Validate size
    if f.Header.Size > maxFileSize {
        return fmt.Errorf("file %s exceeds maximum size of 5 MB", f.FileType)
    }

    // ... storage write and document record upsert follow
}
```

What happens if someone sends a 50 MB file: the `multipart.FileHeader.Size` check catches it before we read the file body or write anything to disk. The function returns an error, the handler sends back HTTP 400 with a clear message like `"file pan exceeds maximum size of 5 MB"`, and the file never reaches storage. The `ParseMultipartForm` call in the handler is bounded to 32 MB of memory, so Go's multipart parser itself also provides a backstop.

The MIME check uses a whitelist map, not a blacklist. If the content type isn't one of the three allowed values, it's rejected. I read the Content-Type header from the multipart form, which is what the browser sends. For a production system I'd also want to sniff the first few bytes with `http.DetectContentType` to catch spoofed headers, but for this scope the header check does the job.

---

## 3. The Queue

The queue is powered by two queries. First, the list query in `store/submission_store.go`:

```go
// Count total submissions in queue
`SELECT COUNT(*) FROM kyc_submissions WHERE state = $1`

// Fetch page of submissions, oldest first
`SELECT id, merchant_id, state, personal_details, business_details, reviewer_note, created_at, updated_at
 FROM kyc_submissions WHERE state = $1 ORDER BY created_at ASC LIMIT $2 OFFSET $3`
```

The SLA flag is computed in Go, not in SQL:

```go
for i, sub := range subs {
    isAtRisk := time.Since(sub.CreatedAt) > 24*time.Hour
    items[i] = models.SubmissionQueueItem{
        SubmissionID: sub.ID,
        MerchantID:   sub.MerchantID,
        State:        sub.State,
        AtRisk:       isAtRisk,
        CreatedAt:    sub.CreatedAt,
    }
}
```

Why I wrote it this way: the `at_risk` flag is a derived value. Storing it in the database would mean it goes stale the moment clock time passes 24 hours. Computing it on every read means it's always accurate. One `time.Since` call per row is negligible.

I filter by `state = 'submitted'` because that's what "in the queue" means per the spec. `ORDER BY created_at ASC` gives oldest-first ordering so reviewers naturally process the longest-waiting submissions. The `LIMIT/OFFSET` pagination is simple and works fine for the queue sizes we're dealing with here.

The metrics at the top of the dashboard come from two more queries:

```sql
-- Average time in queue
SELECT AVG(EXTRACT(EPOCH FROM (now() - created_at))) FROM kyc_submissions WHERE state = 'submitted'

-- Approval rate over last 7 days
SELECT
    COALESCE(SUM(CASE WHEN state = 'approved' THEN 1 ELSE 0 END), 0),
    COALESCE(COUNT(*), 0)
FROM kyc_submissions
WHERE state IN ('approved', 'rejected') AND updated_at >= $1
```

All three metrics are computed at query time. None are cached or stored.

---

## 4. The Auth

Merchant A cannot see merchant B's submission because the system never gives them the option. Here's how:

The merchant endpoints (`/api/v1/kyc/*`) are locked behind `RequireRole("merchant")` middleware. Inside that route group, every handler extracts the `userID` from the JWT context and passes it to the service layer:

```go
// In handlers/kyc_handler.go
func (h *KYCHandler) GetMySubmission(w http.ResponseWriter, r *http.Request) {
    userID := middleware.GetUserID(r.Context())
    sub, err := h.kycService.GetMySubmission(r.Context(), userID)
    // ...
}
```

The service layer then queries `WHERE merchant_id = $1` using that authenticated user ID:

```go
// In store/submission_store.go
`SELECT id, merchant_id, state, ... FROM kyc_submissions WHERE merchant_id = $1`
```

There is no endpoint where a merchant passes in a submission ID and gets back arbitrary data. The `/kyc/me` endpoint always scopes to the logged-in merchant. The `/kyc/save-draft` and `/kyc/submit` endpoints also use `merchantID` from context. A merchant literally cannot reference another merchant's submission because the API doesn't accept a submission ID parameter on merchant routes.

The reviewer endpoints (`/api/v1/reviewer/*`) are behind `RequireRole("reviewer")` middleware. If a merchant's JWT has `role: "merchant"`, the `RequireRole` middleware returns HTTP 403 before the handler even runs:

```go
func RequireRole(role string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            userRole := GetRole(r.Context())
            if userRole != role {
                writeError(w, http.StatusForbidden, "access denied: requires "+role+" role")
                return
            }
            next.ServeHTTP(w, r.WithContext(r.Context()))
        })
    }
}
```

So: merchants are scoped to their own data by query, and role enforcement prevents them from hitting reviewer routes at all.

---

## 5. The AI Audit

While building the `UpdateSubmissionDetails` function in `store/submission_store.go`, the AI generated a version that always set both `personal_details` and `business_details`, even when only one was provided in the request. It looked like this:

```go
// What AI gave me (broken)
func (s *SubmissionStore) UpdateSubmissionDetails(ctx context.Context, id uuid.UUID, pd *models.PersonalDetails, bd *models.BusinessDetails) error {
    pdJSON, _ := json.Marshal(pd)
    bdJSON, _ := json.Marshal(bd)
    _, err := s.db.Pool.Exec(ctx,
        `UPDATE kyc_submissions SET personal_details = $1, business_details = $2, updated_at = now() WHERE id = $3`,
        pdJSON, bdJSON, id)
    return err
}
```

The problem: if a merchant saves a draft with just their personal details (no business details yet), this would marshal the nil `bd` pointer to `null` and overwrite whatever business details were already stored. The save-draft endpoint is explicitly a partial update. Sending only one section should not nuke the other.

What I replaced it with:

```go
func (s *SubmissionStore) UpdateSubmissionDetails(ctx context.Context, id uuid.UUID, personalDetails *models.PersonalDetails, businessDetails *models.BusinessDetails) error {
    query := `UPDATE kyc_submissions SET updated_at = now()`
    args := []interface{}{}
    argIdx := 1

    if personalDetails != nil {
        pdJSON, err := json.Marshal(personalDetails)
        if err != nil {
            return fmt.Errorf("failed to marshal personal details: %w", err)
        }
        query += fmt.Sprintf(", personal_details = $%d", argIdx)
        args = append(args, pdJSON)
        argIdx++
    }
    if businessDetails != nil {
        bdJSON, err := json.Marshal(businessDetails)
        if err != nil {
            return fmt.Errorf("failed to marshal business details: %w", err)
        }
        query += fmt.Sprintf(", business_details = $%d", argIdx)
        args = append(args, bdJSON)
        argIdx++
    }

    query += fmt.Sprintf(" WHERE id = $%d", argIdx)
    args = append(args, id)

    _, err := s.db.Pool.Exec(ctx, query, args...)
    return err
}
```

This dynamically builds the SQL so that only the fields actually provided get updated. If `personalDetails` is nil, it's not in the query at all. This also handles error returns from `json.Marshal` properly instead of swallowing them with `_`.

This is the kind of bug that passes every test where you always send both fields, but breaks the moment a real user saves a half-filled form. The AI treated it like a full-object write; I caught it because I was thinking about how the save-draft flow actually works.
