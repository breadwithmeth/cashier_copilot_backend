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

	"cashier_copilot_backend/internal/config"
	"cashier_copilot_backend/internal/handler"
	"cashier_copilot_backend/internal/model"
	"cashier_copilot_backend/internal/repository"
	"cashier_copilot_backend/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
)

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	logger.Info("starting Cashier Copilot Backend...")

	// --- 1. Load Configuration ---
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}
	logger.Info("configuration loaded",
		"port", cfg.ServerPort,
		"poll_cv_ms", cfg.PollIntervalCvMs,
		"poll_tasks_ms", cfg.PollIntervalTasksMs,
		"confidence_threshold", cfg.ConfidenceThreshold,
		"max_db_conns", cfg.MaxDBConns,
	)

	// --- 2. Create Database Connection Pool ---
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := repository.NewPool(ctx, cfg.DatabaseURL, cfg.MaxDBConns)
	if err != nil {
		logger.Error("failed to create database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// --- 3. Run Database Migrations ---
	if err := repository.RunMigrations(ctx, pool); err != nil {
		logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// --- 4. Initialize Repositories ---
	posRepo := repository.NewPosEventRepo(pool)
	cvRepo := repository.NewCvEventRepo(pool)
	speechRepo := repository.NewSpeechRepo(pool)
	violationRepo := repository.NewViolationRepo(pool)
	taskRepo := repository.NewTaskRepo(pool)
	cameraRepo := repository.NewCameraRepo(pool)
	upsellRepo := repository.NewUpsellRepo(pool)
	userRepo := repository.NewUserRepo(pool)

	// --- 5. Initialize Services ---
	fsmManager := service.NewFSMManager(logger)
	hub := service.NewHub(logger)
	authService := service.NewAuthService(
		userRepo,
		cfg.JWTSecret,
		time.Duration(cfg.AccessTokenTTLMinutes)*time.Minute,
		logger,
	)
	if err := authService.EnsureBootstrapAdmin(ctx, cfg.BootstrapAdminUsername, cfg.BootstrapAdminPassword); err != nil {
		logger.Error("failed to ensure bootstrap admin", "error", err)
		os.Exit(1)
	}

	ruleEngine := service.NewRuleEngine(
		posRepo, cvRepo, speechRepo, violationRepo, taskRepo, cameraRepo,
		fsmManager, hub, cfg.ConfidenceThreshold, logger,
	)

	coPilot := service.NewCoPilot(upsellRepo, speechRepo, fsmManager, hub, logger)

	poller := service.NewPoller(
		cvRepo, speechRepo, taskRepo, cameraRepo, violationRepo,
		ruleEngine, coPilot, fsmManager, hub,
		cfg.PollIntervalCvMs, cfg.PollIntervalTasksMs, logger,
	)

	// --- 6. Start Background Pollers ---
	poller.StartAll(ctx)

	// --- 7. Initialize HTTP Handlers ---
	authHandler := handler.NewAuthHandler(authService, logger)
	userHandler := handler.NewUserHandler(authService, logger)
	posHandler := handler.NewPosHandler(posRepo, fsmManager, ruleEngine, coPilot, logger)
	violationHandler := handler.NewViolationHandler(violationRepo, logger)
	cameraHandler := handler.NewCameraHandler(cameraRepo, logger)
	wsHandler := handler.NewWSHandler(hub, authService, logger)

	// --- 8. Configure Router ---
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/health"))

	// CORS — allow all origins for development
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-API-Key", "X-Request-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	})
	r.Use(corsHandler.Handler)

	// --- 9. Register Routes ---

	// REST API
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", authHandler.HandleLogin)

		r.Group(func(r chi.Router) {
			r.Use(handler.RequireAuth(authService))
			r.Get("/auth/me", authHandler.HandleMe)
		})

		// POS events from 1C terminals
		r.With(handler.RequireAPIKey(cfg.PosAPIKey)).Post("/pos/event", posHandler.HandlePosEvent)

		// Violations journal
		r.With(handler.RequireAuth(authService, model.RoleAdmin, model.RoleOperator)).Get("/violations", violationHandler.HandleListViolations)

		// Camera configuration
		r.With(handler.RequireAuth(authService, model.RoleAdmin)).Post("/cameras", cameraHandler.HandleCreateCamera)
		r.With(handler.RequireAuth(authService, model.RoleAdmin, model.RoleOperator)).Get("/cameras", cameraHandler.HandleListCameras)
		r.With(handler.RequireAuth(authService, model.RoleAdmin, model.RoleOperator)).Get("/cameras/{id}/streams", cameraHandler.HandleGetCameraStreams)
		r.With(handler.RequireAuth(authService, model.RoleAdmin)).Patch("/cameras/{id}/streams", cameraHandler.HandleUpdateCameraStreams)

		// Analytics service callbacks
		r.With(handler.RequireAPIKey(cfg.AnalyticsAPIKey)).Post("/analytics/cameras/{id}/stream", cameraHandler.HandleUpdateCameraStreams)

		// User management
		r.With(handler.RequireAuth(authService, model.RoleAdmin)).Get("/users", userHandler.HandleListUsers)
		r.With(handler.RequireAuth(authService, model.RoleAdmin)).Post("/users", userHandler.HandleCreateUser)
	})

	// WebSocket endpoints
	r.Get("/ws/operator", wsHandler.HandleOperatorWS)
	r.Get("/ws/cashier", wsHandler.HandleCashierWS)

	// --- 10. Start HTTP Server ---
	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	server := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("HTTP server starting", "address", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	logger.Info("Cashier Copilot Backend is running",
		"address", addr,
		"endpoints", []string{
			"POST /api/v1/pos/event",
			"POST /api/v1/auth/login",
			"GET  /api/v1/auth/me",
			"GET  /api/v1/users",
			"POST /api/v1/users",
			"GET  /api/v1/violations",
			"POST /api/v1/cameras",
			"GET  /api/v1/cameras",
			"GET  /api/v1/cameras/{id}/streams",
			"PATCH /api/v1/cameras/{id}/streams",
			"POST /api/v1/analytics/cameras/{id}/stream",
			"GET  /ws/operator",
			"GET  /ws/cashier?pos_id=XXX",
			"GET  /health",
		},
	)

	// --- 11. Graceful Shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info("shutdown signal received", "signal", sig)

	// Cancel context to stop pollers
	cancel()

	// Shutdown HTTP server with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	} else {
		logger.Info("HTTP server stopped gracefully")
	}

	// Close database pool
	pool.Close()
	logger.Info("database pool closed")

	logger.Info("Cashier Copilot Backend shutdown complete")
}
