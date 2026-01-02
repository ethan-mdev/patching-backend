package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	authmiddleware "github.com/ethan-mdev/central-auth/middleware"
	"github.com/ethan-mdev/patching-backend/internal/config"
	"github.com/ethan-mdev/patching-backend/internal/handlers"
	"github.com/ethan-mdev/patching-backend/internal/manifest"
)

func main() {
	// Setup logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load environment variables", "error", err)
		os.Exit(1)
	}
	slog.Info("starting patching server")

	// Load manifest
	m, err := manifest.LoadManifest("./files")
	if err != nil {
		slog.Error("failed to load manifest", "error", err)
		os.Exit(1)
	}
	slog.Info("loaded manifest", "version", m.Version, "files", len(m.Files))

	h := handlers.NewPatchHandler(m, cfg.FilesDir)

	// Initialize JWKS authentication
	ctx := context.Background()
	auth, err := authmiddleware.NewJWKSAuth(ctx, cfg.JWKSUrl, cfg.JWKSRefresh)
	if err != nil {
		slog.Error("failed to create JWKSAuth", "error", err)
		os.Exit(1)
	}
	slog.Info("JWKS authentication configured", "url", cfg.JWKSUrl)

	// Setup router
	mux := http.NewServeMux()

	// Health check (no auth required)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// Protected routes using JWKS verification
	mux.HandleFunc("GET /manifest", h.GetManifest)
	mux.Handle("GET /files/{path...}", auth.Auth(http.HandlerFunc(h.DownloadFile)))

	// Optional debugging endpoint (not used in normal flow)
	mux.Handle("POST /verify", auth.Auth(http.HandlerFunc(h.VerifyFiles)))

	// Admin-only routes (called via CLI or server-to-server)
	mux.Handle("POST /patches/{version}",
		auth.Auth(
			authmiddleware.RequireRole("admin")(
				http.HandlerFunc(h.CreatePatch),
			),
		),
	)

	// Setup server with graceful shutdown
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		slog.Info("server running", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server exited")
}
