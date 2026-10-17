package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type errorResponse struct {
	Error string `json:"error"`
}

type createQuoteRequest struct {
	Text   string `json:"text"`
	Author string `json:"author"`
}

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx := context.Background()
	pool, err := connectDB(ctx, databaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	appName := os.Getenv("APP_NAME")
	if appName == "" {
		appName = os.Getenv("DOCKER_ENV")
	}
	if appName == "" {
		appName = "Web App Dev"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		fmt.Fprintf(w, "Hello from docker env: %s\n", appName)
	})
	mux.HandleFunc("/api/quotes", quotesHandler(pool))

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func quotesHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		switch r.Method {
		case http.MethodGet:
			quotes, err := listQuotes(ctx, pool)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to list quotes")
				return
			}
			writeJSON(w, http.StatusOK, quotes)
		case http.MethodPost:
			var body createQuoteRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, http.StatusBadRequest, "invalid JSON body")
				return
			}
			body.Text = strings.TrimSpace(body.Text)
			body.Author = strings.TrimSpace(body.Author)
			if body.Text == "" {
				writeError(w, http.StatusBadRequest, "text is required")
				return
			}
			q, err := createQuote(ctx, pool, body.Text, body.Author)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to create quote")
				return
			}
			writeJSON(w, http.StatusCreated, q)
		default:
			methodNotAllowed(w)
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}
