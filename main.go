package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3" // SQLite driver

	"github.com/stevemurr/simple-sync-server/handler"
	"github.com/stevemurr/simple-sync-server/store"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// corsMiddleware wraps an http.Handler with CORS headers.
func corsMiddleware(next http.Handler, origins string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origins)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	host := env("HOST", "0.0.0.0")
	port := env("PORT", "8080")
	dataDir := env("DATA_DIR", "./data")
	backend := env("STORE_BACKEND", "json")
	origins := env("ALLOWED_ORIGINS", "*")

	// Handle multiple origins - use first one for the header
	// (for full multi-origin support, check Origin header at request time)
	origin := strings.Split(origins, ",")[0]

	s, err := store.New(backend, dataDir)
	if err != nil {
		log.Fatalf("failed to create store (backend=%s): %v", backend, err)
	}

	h := handler.New(s)
	wrapped := corsMiddleware(h, origin)

	addr := fmt.Sprintf("%s:%s", host, port)
	log.Printf("Simple Sync Server starting on %s (store=%s, data=%s)", addr, backend, dataDir)
	if err := http.ListenAndServe(addr, wrapped); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
