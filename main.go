package main

import (
	"context"
	"log"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
)

func main() {
	// JWKs URL
	jwksURL := "http://localhost:8080/.well-known/jwks.json"

	// JWKs cache
	cache := jwk.NewCache(context.Background())
	cache.Register(jwksURL, jwk.WithMinRefreshInterval(15*time.Minute))

	// Fetch JWKs
	_, err := cache.Refresh(context.Background(), jwksURL)
	if err != nil {
		log.Fatalf("failed to fetch JWKs: %v", err)
	}

	log.Println("Successfully fetched JWKs from", jwksURL)
}
