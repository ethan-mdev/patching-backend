package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/ethan-mdev/central-auth/middleware"
)

func main() {
	// JWKs URL
	jwksURL := "http://localhost:8080/.well-known/jwks.json"

	ctx := context.Background()

	auth, err := middleware.NewJWKSAuth(ctx, jwksURL, 15*time.Minute)
	if err != nil {
		log.Fatalf("failed to create JWKSAuth: %v", err)
	}

	mux := http.NewServeMux()

	// Protected routes using JWKS verification
	mux.Handle("GET /patches", auth.Auth(http.HandlerFunc(listPatches)))

	// With role check
	mux.Handle("POST /patches", auth.Auth(middleware.RequireRole("admin")(http.HandlerFunc(createPatch))))

	log.Println("Server running on :8081")
	http.ListenAndServe(":8081", mux)

	log.Println("Successfully fetched JWKs from", jwksURL)
}

func listPatches(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.GetClaims(r.Context())
	w.Write([]byte("Hello, " + claims.Username))
}

func createPatch(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Patch created"))
}
