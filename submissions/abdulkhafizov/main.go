package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

var startedAt = time.Now()
var db *sql.DB
var jwtSecret string

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	sqlitePath := os.Getenv("SQLITE_PATH")
	jwtSecret = os.Getenv("JWT_SECRET")
	host := envOrDefault("HOST", "127.0.0.1")
	port := envOrDefault("PORT", "3000")

	if sqlitePath == "" {
		log.Fatal("SQLITE_PATH is required")
	}
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	var err error
	db, err = sql.Open("sqlite", sqlitePath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", HealthHandler)
	mux.HandleFunc("GET /feed", FeedHandler)
	mux.HandleFunc("GET /posts/{id}", GetPostHandler)
	mux.HandleFunc("POST /posts", CreatePostHandler)
	mux.HandleFunc("POST /posts/{id}/like", LikePostHandler)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	})

	server := &http.Server{
		Addr:              host + ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	log.Printf("listening on %s\n", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

