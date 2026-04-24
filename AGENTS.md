# AGENTS.md

## 1. Product Overview

This system is a merchant onboarding (KYC) workflow service.

It allows:

* Merchants to submit personal, business, and document information
* Reviewers to evaluate submissions and approve/reject them
* The system to enforce a strict, controlled lifecycle for each submission

The core of the system is a **state-driven workflow**, not CRUD. Every submission must follow a predefined set of transitions, and invalid transitions must be rejected at the API level.

---

## 2. Core Concepts

### 2.1 KYC Submission

A submission represents a merchant's onboarding application.

It contains:

* Personal details (name, email, phone)
* Business details (name, type, expected monthly volume)
* Uploaded documents (PAN, Aadhaar, bank statement)
* Current state
* Timestamps

---

### 2.2 User Roles

#### Merchant

* Can create and edit their own submission
* Can save progress (draft)
* Can submit for review
* Can resubmit after feedback

#### Reviewer

* Can view all submissions
* Can review submissions
* Can approve, reject, or request more information

---

### 2.3 State Machine (Critical)

States:

* `draft`
* `submitted`
* `under_review`
* `approved`
* `rejected`
* `more_info_requested`

Allowed transitions:

* `draft` → `submitted`
* `submitted` → `under_review`
* `under_review` → `approved`
* `under_review` → `rejected`
* `under_review` → `more_info_requested`
* `more_info_requested` → `submitted`

Constraints:

* State transitions must be enforced centrally (single source of truth)
* Any illegal transition must return HTTP 400
* No direct state mutation outside the state machine logic

#### Implementation Pattern (Go)

Define a single transition map and a `Transition` function. No `if/else` chains, no switch statements scattered across handlers.

```go
// services/statemachine.go

type State string

const (
    StateDraft             State = "draft"
    StateSubmitted         State = "submitted"
    StateUnderReview       State = "under_review"
    StateApproved          State = "approved"
    StateRejected          State = "rejected"
    StateMoreInfoRequested State = "more_info_requested"
)

var allowedTransitions = map[State][]State{
    StateDraft:             {StateSubmitted},
    StateSubmitted:         {StateUnderReview},
    StateUnderReview:       {StateApproved, StateRejected, StateMoreInfoRequested},
    StateMoreInfoRequested: {StateSubmitted},
}

// Transition validates and returns the new state, or an error.
// This is the ONLY place a state change is authorized.
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

* All service methods that change state must call `Transition(current, target)` first.
* If it returns an error, the service returns HTTP 400 to the handler — no exception.
* Tests must exercise `Transition` directly, not just via HTTP.

---

## 3. System Architecture

### High-Level Flow

```
Client (React) → API (Go) → Service Layer → Repository → PostgreSQL
                                         → File Storage (local / S3)
```

---

### 3.1 Backend

Language: Go

Structure:

```
/cmd/server         → main.go, app bootstrap
/handlers           → HTTP layer only, no business logic
/services           → business logic, state machine
/store              → database access (repository pattern)
/middleware         → auth, logging, recovery
/models             → shared structs (Submission, User, Document, etc.)
/config             → env/config loading
```

Constraints:

* Business logic MUST NOT exist in handlers
* State machine MUST exist in `services/statemachine.go` only
* Handlers call services; services call store; store calls DB
* No service imports a handler package; no store imports a service package

---

### 3.2 Frontend

* React
* Tailwind CSS

Requirements:

* Multi-step KYC form
* Save draft functionality
* Submission flow
* Reviewer dashboard (queue + detail view)
* Read JWT from `localStorage`, attach as `Authorization: Bearer <token>` on every request

UI is not graded heavily. Focus on correctness and wiring to real API endpoints.

---

### 3.3 Database

Use PostgreSQL.

#### Schema

**users**

| Column       | Type        | Notes                        |
|--------------|-------------|------------------------------|
| id           | UUID PK     | `gen_random_uuid()`          |
| email        | TEXT UNIQUE |                              |
| password     | TEXT        | bcrypt hash                  |
| role         | TEXT        | `merchant` or `reviewer`     |
| created_at   | TIMESTAMPTZ |                              |

**kyc_submissions**

| Column            | Type        | Notes                                          |
|-------------------|-------------|------------------------------------------------|
| id                | UUID PK     |                                                |
| merchant_id       | UUID FK     | → users.id                                     |
| state             | TEXT        | enum-like, enforced at app layer               |
| personal_details  | JSONB       | see §3.3.1                                     |
| business_details  | JSONB       | see §3.3.1                                     |
| reviewer_note     | TEXT        | nullable; set on rejection or more_info        |
| created_at        | TIMESTAMPTZ |                                                |
| updated_at        | TIMESTAMPTZ |                                                |

**documents**

| Column           | Type    | Notes                                              |
|------------------|---------|----------------------------------------------------|
| id               | UUID PK |                                                    |
| submission_id    | UUID FK | → kyc_submissions.id                               |
| file_type        | TEXT    | `pan`, `aadhaar`, `bank_statement`                 |
| storage_key      | TEXT    | relative path or S3 object key — never absolute    |
| storage_backend  | TEXT    | `local` or `s3`                                    |
| original_name    | TEXT    | original filename from upload                      |
| mime_type        | TEXT    |                                                    |
| size_bytes       | BIGINT  |                                                    |
| uploaded_at      | TIMESTAMPTZ |                                                |

**notifications**

| Column       | Type        | Notes                                    |
|--------------|-------------|------------------------------------------|
| id           | UUID PK     |                                          |
| merchant_id  | UUID FK     | → users.id                               |
| event_type   | TEXT        | `submitted`, `approved`, `rejected`, etc.|
| payload      | JSONB       | arbitrary context (reviewer note, etc.)  |
| created_at   | TIMESTAMPTZ |                                          |

#### 3.3.1 JSONB Detail Schemas

These are Go structs that map to the JSONB columns. Unmarshal strictly — unknown fields must error.

```go
type PersonalDetails struct {
    FullName string `json:"full_name"`
    Email    string `json:"email"`
    Phone    string `json:"phone"`
}

type BusinessDetails struct {
    BusinessName          string  `json:"business_name"`
    BusinessType          string  `json:"business_type"`
    ExpectedMonthlyVolume float64 `json:"expected_monthly_volume"`
}
```

Validation must run on these structs in the service layer before any DB write.

---

### 3.4 File Storage

**Abstraction**

Define a `StorageBackend` interface. Handlers and services never reference local disk or S3 directly.

```go
type StorageBackend interface {
    Save(ctx context.Context, key string, r io.Reader) error
    URL(key string) string
    Delete(ctx context.Context, key string) error
}
```

Provide two implementations: `LocalStorage` (writes to a configured root dir) and `S3Storage` (uses AWS SDK). Select via config.

**Key Format**

```
submissions/{submission_id}/{file_type}/{uuid}.{ext}
```

This is the value stored in `documents.storage_key`. Never store an absolute path.

**Validation (server-side)**

Checked before storage write:

* MIME type must be one of: `application/pdf`, `image/jpeg`, `image/png`
* File size must not exceed 5 MB
* Reject with HTTP 400 and a descriptive error if either fails

---

## 4. Authentication

### Mechanism

JWT (JSON Web Token), symmetric signing (HS256). Secret loaded from environment variable `JWT_SECRET`.

### Token Claims

```json
{
  "sub": "<user_id>",
  "role": "merchant | reviewer",
  "exp": <unix_timestamp>
}
```

Token TTL: 24 hours.

### Endpoints

```
POST /api/v1/signup   → create user, return JWT
POST /api/v1/login    → verify password, return JWT
```

Password storage: bcrypt, cost factor ≥ 12.

### Middleware

`AuthMiddleware` reads `Authorization: Bearer <token>`, validates signature and expiry, injects `userID` and `role` into request context.

All routes except `/signup` and `/login` require this middleware.

### Authorization Rules

Enforced in service layer, not handler:

* Merchant: can only read/write their own submission (`submission.merchant_id == ctx.userID`)
* Reviewer: can read all submissions, trigger transitions
* Any violation → HTTP 403

---

## 5. API Design

Base path: `/api/v1/`

All request and response bodies are JSON unless the endpoint handles multipart (file upload).

All error responses follow:

```json
{
  "error": "human-readable message"
}
```

---

### 5.1 Auth APIs

#### POST /api/v1/signup

Request:
```json
{ "email": "...", "password": "...", "role": "merchant | reviewer" }
```

Response `201`:
```json
{ "token": "<jwt>" }
```

---

#### POST /api/v1/login

Request:
```json
{ "email": "...", "password": "..." }
```

Response `200`:
```json
{ "token": "<jwt>" }
```

---

### 5.2 Merchant APIs

All require `Authorization: Bearer <token>` with role `merchant`.

---

#### POST /api/v1/kyc/save-draft

Creates or updates the merchant's submission in `draft` state.

Request: `multipart/form-data`

Fields:

| Field              | Type   | Required |
|--------------------|--------|----------|
| personal_details   | JSON string | No  |
| business_details   | JSON string | No  |
| pan                | file   | No       |
| aadhaar            | file   | No       |
| bank_statement     | file   | No       |

* Only fields present in the request are updated (partial update).
* Submission must be in `draft` state. Any other state → HTTP 400.
* If no submission exists yet, one is created in `draft` state.

Response `200`:
```json
{ "submission_id": "...", "state": "draft" }
```

---

#### POST /api/v1/kyc/submit

Transitions the merchant's submission from `draft` → `submitted`.

* Requires all three documents to be present.
* Requires `personal_details` and `business_details` to be complete and valid.
* On success, creates a notification record (`event_type: submitted`).

Request: `{}` (empty body)

Response `200`:
```json
{ "submission_id": "...", "state": "submitted" }
```

---

#### GET /api/v1/kyc/me

Returns the merchant's own submission with current state and document list.

Response `200`:
```json
{
  "submission_id": "...",
  "state": "...",
  "personal_details": { ... },
  "business_details": { ... },
  "documents": [
    { "file_type": "pan", "storage_key": "...", "uploaded_at": "..." }
  ],
  "reviewer_note": "...",
  "created_at": "...",
  "updated_at": "..."
}
```

---

### 5.3 Reviewer APIs

All require `Authorization: Bearer <token>` with role `reviewer`.

---

#### GET /api/v1/reviewer/queue

Returns submissions in `submitted` state, oldest first.

Query params:

| Param  | Default | Notes              |
|--------|---------|--------------------|
| limit  | 20      | max 100            |
| offset | 0       |                    |

Response `200`:
```json
{
  "submissions": [
    {
      "submission_id": "...",
      "merchant_id": "...",
      "state": "submitted",
      "at_risk": true,
      "created_at": "..."
    }
  ],
  "total": 42
}
```

`at_risk` is `true` if `now - created_at > 24h`. Computed in Go, never stored.

---

#### GET /api/v1/reviewer/:id

Returns full submission detail including documents.

Response `200`: same shape as `GET /kyc/me` but accessible for any submission ID.

---

#### POST /api/v1/reviewer/:id/transition

Triggers a state transition on a submission.

Request:
```json
{
  "to": "approved | rejected | under_review | more_info_requested",
  "note": "optional reviewer note"
}
```

* Calls `Transition(current, to)` — if error, HTTP 400.
* `note` is required when `to` is `rejected` or `more_info_requested`.
* On success, creates a notification record.

Response `200`:
```json
{ "submission_id": "...", "state": "..." }
```

---

### 5.4 Metrics API

Requires `Authorization: Bearer <token>` with role `reviewer`.

#### GET /api/v1/metrics

Response `200`:
```json
{
  "queue_size": 12,
  "avg_time_in_queue_seconds": 43200,
  "approval_rate_last_7d": 0.73
}
```

Definitions:

* `queue_size`: count of submissions in `submitted` state
* `avg_time_in_queue_seconds`: average of `(now - created_at)` for submissions in `submitted` state
* `approval_rate_last_7d`: `approved_count / total_finalized_count` for submissions that reached a terminal state (`approved` or `rejected`) within the last 7 days

All three computed at query time. None stored in DB.

---

## 6. Validation Rules

### 6.1 File Upload

* Allowed MIME types: `application/pdf`, `image/jpeg`, `image/png`
* Max size: 5 MB
* Validated server-side before writing to storage
* Invalid type or size → HTTP 400 with specific message

### 6.2 State Transitions

* All transitions go through `services/statemachine.go:Transition()`
* Invalid transition → HTTP 400: `"invalid transition: \"approved\" → \"draft\" is not allowed"`

### 6.3 Authorization

* Merchant accessing another merchant's submission → HTTP 403
* Merchant calling reviewer endpoints → HTTP 403
* Unauthenticated request to any protected route → HTTP 401

### 6.4 Input Validation

* All required JSON fields checked before DB write
* Empty or missing `full_name`, `email`, `phone` in `personal_details` → HTTP 400
* `expected_monthly_volume` must be > 0
* Email format validated with regex

---

## 7. Notifications

On every state change, insert one row into `notifications`:

```go
type Notification struct {
    ID         uuid.UUID              `db:"id"`
    MerchantID uuid.UUID              `db:"merchant_id"`
    EventType  string                 `db:"event_type"`
    Payload    map[string]interface{} `db:"payload"`
    CreatedAt  time.Time              `db:"created_at"`
}
```

`event_type` mirrors the target state: `submitted`, `under_review`, `approved`, `rejected`, `more_info_requested`.

`payload` should include at minimum:

```json
{ "submission_id": "...", "reviewer_note": "..." }
```

No external delivery. Stored for in-app polling or future webhook use.

---

## 8. SLA Tracking

* A submission in `submitted` state for more than 24 hours is `at_risk`.
* Computed: `time.Since(submission.CreatedAt) > 24*time.Hour`
* Never stored as a column. Computed on every read in the service layer.
* Surfaced in `GET /reviewer/queue` response as `"at_risk": true/false`.

---

## 9. Error Handling

All errors return:

```json
{ "error": "descriptive message" }
```

| Scenario                     | HTTP Status |
|------------------------------|-------------|
| Invalid state transition      | 400         |
| Validation failure            | 400         |
| Invalid file type/size        | 400         |
| Unauthenticated               | 401         |
| Unauthorized (wrong role/owner) | 403       |
| Resource not found            | 404         |
| Internal server error         | 500         |

HTTP 500 responses must not leak internal error detail to the client. Log internally, return generic message.

---

## 10. Testing Requirements

Minimum required test coverage:

1. **Illegal state transition** — call `Transition("approved", "draft")`, assert error returned, verify HTTP 400 from the handler.
2. **Authorization** — merchant attempting to access another merchant's submission, assert HTTP 403.
3. **File validation** — upload a file exceeding 5 MB or with disallowed MIME type, assert HTTP 400.
4. **Happy path** — merchant creates draft, submits, reviewer transitions to `under_review`, then `approved`. Assert final state is `approved` and notification records exist.

Tests must use a real PostgreSQL instance (test DB) or `sqlc`-compatible mock. No skipping DB layer.

---

## 11. Seed Data

Run on startup if DB is empty.

Seed script must create:

* 1 reviewer account: `reviewer@kyc.dev` / `password123`
* 2 merchant accounts:
  * `merchant.draft@kyc.dev` / `password123` — submission in `draft` state with partial personal details
  * `merchant.review@kyc.dev` / `password123` — submission in `under_review` state with all documents present

All passwords stored as bcrypt hashes. Seed must be idempotent (check before insert).

---

## 12. Configuration

All config via environment variables. No hardcoded values.

| Variable         | Required | Default       | Notes                          |
|------------------|----------|---------------|--------------------------------|
| DATABASE_URL     | Yes      | —             | PostgreSQL connection string   |
| JWT_SECRET       | Yes      | —             | Min 32 chars                   |
| STORAGE_BACKEND  | No       | `local`       | `local` or `s3`                |
| STORAGE_ROOT     | No       | `./uploads`   | Used when `STORAGE_BACKEND=local` |
| AWS_BUCKET       | No       | —             | Required if `STORAGE_BACKEND=s3` |
| AWS_REGION       | No       | —             |                                |
| PORT             | No       | `8080`        |                                |

---

## 13. Non-Goals

Do NOT implement:

* Payment processing
* Email or SMS delivery
* OAuth or third-party auth
* Microservices or distributed systems
* Real-time WebSocket notifications

---

## 14. Implementation Principles

* State machine is the core abstraction — build it first, test it in isolation
* Centralize business logic in service layer — handlers are thin
* Validate everything server-side — never trust client input
* Storage abstraction behind interface — no local disk references in business logic
* Prefer clarity over cleverness
* Working system > over-engineered system

---

## 15. Optional Enhancements (If Time Permits)

* Audit log table: `(id, submission_id, from_state, to_state, actor_id, created_at)` — append-only, written on every transition
* Reviewer assignment: round-robin across reviewer pool, stored in `kyc_submissions.assigned_reviewer_id`
* Docker Compose setup: `postgres`, `backend`, `frontend` services with volume for uploads
* Drag-and-drop file upload in React frontend