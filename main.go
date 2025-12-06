package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/ethan-mdev/central-auth/middleware"
	"github.com/ethan-mdev/patching-backend/internal/handlers"
	"github.com/ethan-mdev/patching-backend/internal/manifest"
)

const filesDir = "./files"

func main() {
	m, err := manifest.LoadManifest(".")
	if err != nil {
		log.Fatalf("failed to load manifest: %v", err)
	}
	log.Printf("Loaded manifest version: %s with %d files", m.Version, len(m.Files))

	h := handlers.NewPatchHandler(m, filesDir)

	// JWKs URL
	jwksURL := "http://localhost:8080/.well-known/jwks.json"

	ctx := context.Background()

	auth, err := middleware.NewJWKSAuth(ctx, jwksURL, 15*time.Minute)
	if err != nil {
		log.Fatalf("failed to create JWKSAuth: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// Protected routes using JWKS verification
	mux.Handle("GET /manifest", auth.Auth(http.HandlerFunc(h.GetManifest)))
	mux.Handle("GET /version", auth.Auth(http.HandlerFunc(h.GetVersion)))
	mux.Handle("GET /files/{path...}", auth.Auth(http.HandlerFunc(h.DownloadFile)))
	mux.Handle("GET /files/batch", auth.Auth(http.HandlerFunc(h.DownloadBatch)))

	// With role check
	mux.Handle("POST /patches/{version}", auth.Auth(middleware.RequireRole("admin")(http.HandlerFunc(h.CreatePatch))))

	log.Println("Server running on :8081")
	http.ListenAndServe(":8081", mux)
}
