package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"kyc/backend/handlers"
	"kyc/backend/middleware"
	"kyc/backend/services"
	"kyc/backend/storage"
	"kyc/backend/store"
)

// TestEnv holds all shared test infrastructure.
type TestEnv struct {
	Pool             *pgxpool.Pool
	DB               *store.DB
	UserStore        *store.UserStore
	SubmissionStore  *store.SubmissionStore
	DocumentStore   *store.DocumentStore
	NotificationStore *store.NotificationStore
	AuthService      *services.AuthService
	KYCService       *services.KYCService
	ReviewerService  *services.ReviewerService
	MetricsService   *services.MetricsService
	Storage          storage.StorageBackend
	Router           chi.Router
}

const TestJWTSecret = "test-jwt-secret-must-be-at-least-32-characters-long"

// SetupTestDB connects to the test database, runs migrations, and returns a cleanup func.
func SetupTestDB(t *testing.T) *TestEnv {
	t.Helper()

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration tests")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect to test DB: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("failed to ping test DB: %v", err)
	}

	// Run migrations
	migrationSQL, err := os.ReadFile("../migrations/001_init.sql")
	if err != nil {
		// Try from project root
		migrationSQL, err = os.ReadFile("migrations/001_init.sql")
		if err != nil {
			t.Fatalf("failed to read migration file: %v", err)
		}
	}

	if _, err := pool.Exec(ctx, string(migrationSQL)); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	db := &store.DB{Pool: pool}

	// Create stores
	userStore := store.NewUserStore(db)
	submissionStore := store.NewSubmissionStore(db)
	documentStore := store.NewDocumentStore(db)
	notificationStore := store.NewNotificationStore(db)

	// Create storage
	tmpDir := t.TempDir()
	storageBackend, err := storage.NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to create test storage: %v", err)
	}

	// Create services
	authService := services.NewAuthService(userStore, TestJWTSecret)
	kycService := services.NewKYCService(submissionStore, documentStore, notificationStore, storageBackend, "local")
	reviewerService := services.NewReviewerService(submissionStore, documentStore, notificationStore)
	metricsService := services.NewMetricsService(submissionStore)

	// Build router
	authHandler := handlers.NewAuthHandler(authService)
	kycHandler := handlers.NewKYCHandler(kycService)
	reviewerHandler := handlers.NewReviewerHandler(reviewerService)
	metricsHandler := handlers.NewMetricsHandler(metricsService)

	r := chi.NewRouter()
	r.Use(chiMiddleware.RequestID)
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/signup", authHandler.Signup)
		r.Post("/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(authService))

			r.Route("/kyc", func(r chi.Router) {
				r.Use(middleware.RequireRole("merchant"))
				r.Post("/save-draft", kycHandler.SaveDraft)
				r.Post("/submit", kycHandler.Submit)
				r.Get("/me", kycHandler.GetMySubmission)
			})

			r.Route("/reviewer", func(r chi.Router) {
				r.Use(middleware.RequireRole("reviewer"))
				r.Get("/queue", reviewerHandler.ListQueue)
				r.Get("/{id}", reviewerHandler.GetSubmission)
				r.Post("/{id}/transition", reviewerHandler.TransitionSubmission)
			})

			r.Route("/metrics", func(r chi.Router) {
				r.Use(middleware.RequireRole("reviewer"))
				r.Get("/", metricsHandler.GetMetrics)
			})
		})
	})

	env := &TestEnv{
		Pool:              pool,
		DB:                db,
		UserStore:         userStore,
		SubmissionStore:   submissionStore,
		DocumentStore:     documentStore,
		NotificationStore: notificationStore,
		AuthService:       authService,
		KYCService:        kycService,
		ReviewerService:   reviewerService,
		MetricsService:    metricsService,
		Storage:           storageBackend,
		Router:            r,
	}

	t.Cleanup(func() {
		CleanDB(t, pool)
		pool.Close()
	})

	return env
}

// CleanDB truncates all tables for test isolation.
func CleanDB(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		TRUNCATE notifications, documents, kyc_submissions, users CASCADE;
	`)
	if err != nil {
		t.Logf("warning: failed to clean DB: %v", err)
	}
}

// CreateMerchant creates a merchant user and returns the user, JWT, and a cleanup function.
func CreateMerchant(t *testing.T, env *TestEnv, email string) (uuid.UUID, string) {
	t.Helper()
	return createUser(t, env, email, "merchant")
}

// CreateReviewer creates a reviewer user and returns the user ID and JWT.
func CreateReviewer(t *testing.T, env *TestEnv, email string) (uuid.UUID, string) {
	t.Helper()
	return createUser(t, env, email, "reviewer")
}

func createUser(t *testing.T, env *TestEnv, email, role string) (uuid.UUID, string) {
	t.Helper()
	ctx := context.Background()

	hashed, err := bcrypt.GenerateFromPassword([]byte("password123"), 4) // low cost for tests
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	user, err := env.UserStore.CreateUser(ctx, email, string(hashed), role)
	if err != nil {
		t.Fatalf("failed to create %s %q: %v", role, email, err)
	}

	token, err := env.AuthService.Login(ctx, email, "password123")
	if err != nil {
		t.Fatalf("failed to generate JWT for %q: %v", email, err)
	}

	return user.ID, token
}

// CreateSubmission creates a submission in the given state with full details + docs.
func CreateSubmission(t *testing.T, env *TestEnv, merchantID uuid.UUID, state string) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	sub, err := env.SubmissionStore.CreateSubmission(ctx, merchantID)
	if err != nil {
		t.Fatalf("failed to create submission: %v", err)
	}

	// Add personal + business details
	pd := &struct {
		FullName string `json:"full_name"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
	}{"Test User", "test@test.com", "+91-1234567890"}
	bd := &struct {
		BusinessName          string  `json:"business_name"`
		BusinessType          string  `json:"business_type"`
		ExpectedMonthlyVolume float64 `json:"expected_monthly_volume"`
	}{"Test Corp", "retail", 50000}

	pdJSON, _ := json.Marshal(pd)
	bdJSON, _ := json.Marshal(bd)

	_, err = env.Pool.Exec(ctx,
		`UPDATE kyc_submissions SET personal_details = $1, business_details = $2 WHERE id = $3`,
		pdJSON, bdJSON, sub.ID)
	if err != nil {
		t.Fatalf("failed to update submission details: %v", err)
	}

	// Add 3 documents
	for _, ft := range []string{"pan", "aadhaar", "bank_statement"} {
		_, err = env.Pool.Exec(ctx,
			`INSERT INTO documents (submission_id, file_type, storage_key, storage_backend, original_name, mime_type, size_bytes)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			sub.ID, ft,
			fmt.Sprintf("test/%s/%s/test.pdf", sub.ID, ft),
			"local", ft+".pdf", "application/pdf", 1024)
		if err != nil {
			t.Fatalf("failed to create test document %s: %v", ft, err)
		}
	}

	// Transition to desired state through valid path
	transitions := statePathTo(state)
	for _, target := range transitions {
		var notePtr *string
		if target == "rejected" || target == "more_info_requested" {
			n := "test note"
			notePtr = &n
		}
		if err := env.SubmissionStore.UpdateSubmissionState(ctx, sub.ID, target, notePtr); err != nil {
			t.Fatalf("failed to transition to %s: %v", target, err)
		}
	}

	return sub.ID
}

// CreateSubmissionWithTime creates a submission with a specific created_at timestamp.
func CreateSubmissionWithTime(t *testing.T, env *TestEnv, merchantID uuid.UUID, state string, createdAt time.Time) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	subID := CreateSubmission(t, env, merchantID, state)

	_, err := env.Pool.Exec(ctx,
		`UPDATE kyc_submissions SET created_at = $1 WHERE id = $2`,
		createdAt, subID)
	if err != nil {
		t.Fatalf("failed to set created_at: %v", err)
	}

	return subID
}

// statePathTo returns the sequence of states to transition through to reach the target.
func statePathTo(target string) []string {
	switch target {
	case "draft":
		return nil
	case "submitted":
		return []string{"submitted"}
	case "under_review":
		return []string{"submitted", "under_review"}
	case "approved":
		return []string{"submitted", "under_review", "approved"}
	case "rejected":
		return []string{"submitted", "under_review", "rejected"}
	case "more_info_requested":
		return []string{"submitted", "under_review", "more_info_requested"}
	default:
		return nil
	}
}

// MakeRequest sends an HTTP request to the test router and returns the recorder.
func MakeRequest(t *testing.T, env *TestEnv, method, path string, body interface{}, jwt string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
		reader = bytes.NewReader(b)
	}

	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if jwt != "" {
		req.Header.Set("Authorization", "Bearer "+jwt)
	}

	rr := httptest.NewRecorder()
	env.Router.ServeHTTP(rr, req)
	return rr
}

// MakeMultipartRequest sends a multipart/form-data request.
func MakeMultipartRequest(t *testing.T, env *TestEnv, path string, fields map[string]string, files map[string]FileData, jwt string) *httptest.ResponseRecorder {
	t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for key, val := range fields {
		if err := writer.WriteField(key, val); err != nil {
			t.Fatalf("failed to write field %q: %v", key, err)
		}
	}

	for key, fd := range files {
		part, err := writer.CreateFormFile(key, fd.Filename)
		if err != nil {
			t.Fatalf("failed to create form file %q: %v", key, err)
		}
		if _, err := part.Write(fd.Content); err != nil {
			t.Fatalf("failed to write file content for %q: %v", key, err)
		}
	}

	writer.Close()

	req := httptest.NewRequest("POST", path, &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if jwt != "" {
		req.Header.Set("Authorization", "Bearer "+jwt)
	}

	rr := httptest.NewRecorder()
	env.Router.ServeHTTP(rr, req)
	return rr
}

// FileData holds file content for multipart uploads.
type FileData struct {
	Filename string
	Content  []byte
}

// MakePDFContent creates a minimal valid PDF byte slice of the given size.
func MakePDFContent(size int) []byte {
	header := []byte("%PDF-1.4 test content\n")
	if size <= len(header) {
		return header[:size]
	}
	content := make([]byte, size)
	copy(content, header)
	return content
}

// MakeJPEGContent creates content with a JPEG magic number.
func MakeJPEGContent(size int) []byte {
	header := []byte{0xFF, 0xD8, 0xFF, 0xE0} // JPEG magic bytes
	if size <= len(header) {
		return header[:size]
	}
	content := make([]byte, size)
	copy(content, header)
	return content
}

// MakePNGContent creates content with a PNG magic number.
func MakePNGContent(size int) []byte {
	header := []byte{0x89, 0x50, 0x4E, 0x47} // PNG magic bytes
	if size <= len(header) {
		return header[:size]
	}
	content := make([]byte, size)
	copy(content, header)
	return content
}

// DecodeResponse decodes a JSON response body into the target.
func DecodeResponse(t *testing.T, rr *httptest.ResponseRecorder, target interface{}) {
	t.Helper()
	if err := json.NewDecoder(rr.Body).Decode(target); err != nil {
		t.Fatalf("failed to decode response body: %v (body: %s)", err, rr.Body.String())
	}
}

// AssertStatus asserts the HTTP status code.
func AssertStatus(t *testing.T, rr *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if rr.Code != expected {
		t.Errorf("expected status %d, got %d (body: %s)", expected, rr.Code, rr.Body.String())
	}
}

// AssertErrorContains asserts the response has the given status and contains the substring.
func AssertErrorContains(t *testing.T, rr *httptest.ResponseRecorder, status int, substr string) {
	t.Helper()
	AssertStatus(t, rr, status)
	body := rr.Body.String()
	if !strings.Contains(body, substr) {
		t.Errorf("expected response body to contain %q, got: %s", substr, body)
	}
}

// CountRows returns the number of rows in a table matching the given condition.
func CountRows(t *testing.T, pool *pgxpool.Pool, table, where string, args ...interface{}) int {
	t.Helper()
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	if where != "" {
		query += " WHERE " + where
	}
	var count int
	if err := pool.QueryRow(context.Background(), query, args...).Scan(&count); err != nil {
		t.Fatalf("failed to count rows in %s: %v", table, err)
	}
	return count
}
