package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"kyc/backend/config"
	"kyc/backend/handlers"
	"kyc/backend/middleware"
	"kyc/backend/seed"
	"kyc/backend/services"
	"kyc/backend/storage"
	"kyc/backend/store"
)

func main() {
	// Initialize structured logger (JSON for production, text for dev)
	logLevel := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	slog.Info("starting KYC server",
		"log_level", logLevel.String(),
	)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	slog.Info("configuration loaded",
		"port", cfg.Port,
		"storage_backend", cfg.StorageBackend,
	)

	ctx := context.Background()

	// Connect to database
	db, err := store.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("database connected")

	// Run migrations
	migrationSQL, err := os.ReadFile("migrations/001_init.sql")
	if err != nil {
		slog.Error("failed to read migration file", "error", err)
		os.Exit(1)
	}
	if err := db.RunMigrations(ctx, string(migrationSQL)); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("database migrations complete")

	// Seed data
	if err := seed.Run(ctx, db.Pool); err != nil {
		slog.Error("failed to seed database", "error", err)
		os.Exit(1)
	}

	// Initialize storage backend
	var storageBackend storage.StorageBackend
	switch cfg.StorageBackend {
	case "s3":
		storageBackend, err = storage.NewS3Storage(cfg.AWSBucket, cfg.AWSRegion)
	default:
		storageBackend, err = storage.NewLocalStorage(cfg.StorageRoot)
	}
	if err != nil {
		slog.Error("failed to initialize storage", "backend", cfg.StorageBackend, "error", err)
		os.Exit(1)
	}
	slog.Info("storage backend initialized",
		"backend", cfg.StorageBackend,
		"root", cfg.StorageRoot,
	)

	// Initialize stores
	userStore := store.NewUserStore(db)
	submissionStore := store.NewSubmissionStore(db)
	documentStore := store.NewDocumentStore(db)
	notificationStore := store.NewNotificationStore(db)

	// Initialize services
	authService := services.NewAuthService(userStore, cfg.JWTSecret)
	kycService := services.NewKYCService(submissionStore, documentStore, notificationStore, storageBackend, cfg.StorageBackend)
	reviewerService := services.NewReviewerService(submissionStore, documentStore, notificationStore)
	metricsService := services.NewMetricsService(submissionStore)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	kycHandler := handlers.NewKYCHandler(kycService)
	reviewerHandler := handlers.NewReviewerHandler(reviewerService)
	metricsHandler := handlers.NewMetricsHandler(metricsService)
	notificationHandler := handlers.NewNotificationHandler(notificationStore)

	// Build router
	r := chi.NewRouter()

	// Global middleware
	r.Use(chiMiddleware.RequestID)
	r.Use(middleware.Recovery)
	r.Use(middleware.Logger)
	r.Use(corsMiddleware)

	// Public routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/signup", authHandler.Signup)
		r.Post("/login", authHandler.Login)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(authService))

			// Merchant routes
			r.Route("/kyc", func(r chi.Router) {
				r.Use(middleware.RequireRole("merchant"))
				r.Post("/save-draft", kycHandler.SaveDraft)
				r.Post("/submit", kycHandler.Submit)
				r.Get("/me", kycHandler.GetMySubmission)
				r.Get("/notifications", notificationHandler.GetNotifications)
			})

			// Reviewer routes
			r.Route("/reviewer", func(r chi.Router) {
				r.Use(middleware.RequireRole("reviewer"))
				r.Get("/queue", reviewerHandler.ListQueue)
				r.Get("/{id}", reviewerHandler.GetSubmission)
				r.Post("/{id}/transition", reviewerHandler.TransitionSubmission)
			})

			// Metrics route
			r.Route("/metrics", func(r chi.Router) {
				r.Use(middleware.RequireRole("reviewer"))
				r.Get("/", metricsHandler.GetMetrics)
			})
		})
	})

	// Serve uploaded files
	if cfg.StorageBackend == "local" {
		fileServer := http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.StorageRoot)))
		r.Handle("/uploads/*", fileServer)
	}

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigChan
		slog.Info("shutdown signal received", "signal", sig.String())
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("graceful shutdown failed", "error", err)
		}
	}()

	slog.Info("server starting",
		"addr", addr,
		"routes", []string{
			"POST /api/v1/signup",
			"POST /api/v1/login",
			"POST /api/v1/kyc/save-draft",
			"POST /api/v1/kyc/submit",
			"GET  /api/v1/kyc/me",
			"GET  /api/v1/reviewer/queue",
			"GET  /api/v1/reviewer/{id}",
			"POST /api/v1/reviewer/{id}/transition",
			"GET  /api/v1/metrics/",
		},
	)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}

// corsMiddleware adds CORS headers for frontend development.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type")
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
