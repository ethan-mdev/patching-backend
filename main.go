package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	authmiddleware "github.com/ethan-mdev/central-auth/middleware"
	"github.com/ethan-mdev/patching-backend/internal/config"
	"github.com/ethan-mdev/patching-backend/internal/handlers"
	"github.com/ethan-mdev/patching-backend/internal/manifest"
	"github.com/ethan-mdev/patching-backend/internal/middleware"
)

func main() {
	// Load configuration
	cfg := config.Load()
	log.Printf("Starting patching server in %s mode", cfg.Environment)

	// Load manifest
	m, err := manifest.LoadManifest(".")
	if err != nil {
		log.Fatalf("failed to load manifest: %v", err)
	}
	log.Printf("Loaded manifest version: %s with %d files", m.Version, len(m.Files))

	h := handlers.NewPatchHandler(m, cfg.FilesDir)

	// Initialize JWKS authentication
	ctx := context.Background()
	auth, err := authmiddleware.NewJWKSAuth(ctx, cfg.JWKSUrl, cfg.JWKSRefresh)
	if err != nil {
		log.Fatalf("failed to create JWKSAuth: %v", err)
	}
	log.Printf("JWKS authentication configured with URL: %s", cfg.JWKSUrl)

	// Setup router
	mux := http.NewServeMux()

	// Health check (no auth required)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// Protected routes using JWKS verification
	mux.Handle("GET /manifest", auth.Auth(http.HandlerFunc(h.GetManifest)))
	mux.Handle("GET /version", auth.Auth(http.HandlerFunc(h.GetVersion)))
	mux.Handle("GET /files/{path...}", auth.Auth(http.HandlerFunc(h.DownloadFile)))

	// Optional debugging endpoint (not used in normal flow)
	mux.Handle("POST /verify", auth.Auth(http.HandlerFunc(h.VerifyFiles)))

	// Admin-only routes
	mux.Handle("POST /patches/{version}",
		auth.Auth(
			authmiddleware.RequireRole("admin")(
				http.HandlerFunc(h.CreatePatch),
			),
		),
	)

	limiter := middleware.NewRateLimiter(10) // 10 requests per minute per IP

	// Wrap with middleware
	handler := middleware.Logging(
		limiter.Middleware(
			middleware.Compress(
				middleware.CORS(cfg.AllowedOrigins)(mux),
			),
		),
	)

	// Setup server with graceful shutdown
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server running on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
