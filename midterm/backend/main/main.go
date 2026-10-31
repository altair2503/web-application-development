package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type errorResponse struct {
	Error string `json:"error"`
}

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	pool, err := connectDB(context.Background(), databaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	appName := os.Getenv("APP_NAME")
	if appName == "" {
		appName = "City Insights"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from docker env: %s\n", appName)
	})
	mux.HandleFunc("GET /api/places", listHandler(pool))
	mux.HandleFunc("POST /api/places", createHandler(pool))
	mux.HandleFunc("GET /api/places/{id}", getHandler(pool))
	mux.HandleFunc("PUT /api/places/{id}", updateHandler(pool))
	mux.HandleFunc("DELETE /api/places/{id}", deleteHandler(pool))
	// Method-less fallbacks: ServeMux prefers the method-specific patterns above.
	mux.HandleFunc("/api/places", methodNotAllowed)
	mux.HandleFunc("/api/places/{id}", methodNotAllowed)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func listHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		q := r.URL.Query()
		places, err := listPlaces(ctx, pool, strings.TrimSpace(q.Get("city")), strings.TrimSpace(q.Get("category")))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list places")
			return
		}
		writeJSON(w, http.StatusOK, places)
	}
}

func getHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		p, err := getPlace(ctx, pool, id)
		respond(w, http.StatusOK, p, err, "failed to get place")
	}
}

func createHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := decodePlace(w, r)
		if !ok {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		created, err := createPlace(ctx, pool, p)
		respond(w, http.StatusCreated, created, err, "failed to create place")
	}
}

func updateHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		p, ok := decodePlace(w, r)
		if !ok {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		updated, err := updatePlace(ctx, pool, id, p)
		respond(w, http.StatusOK, updated, err, "failed to update place")
	}
}

func deleteHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		if err := deletePlace(ctx, pool, id); err != nil {
			respond(w, 0, nil, err, "failed to delete place")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func pathID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "id must be a positive integer")
		return 0, false
	}
	return id, true
}

// decodePlace parses and validates a request body (shared by POST and PUT).
func decodePlace(w http.ResponseWriter, r *http.Request) (Place, bool) {
	var p Place
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return p, false
	}
	p.Name = strings.TrimSpace(p.Name)
	p.City = strings.TrimSpace(p.City)
	p.Category = strings.TrimSpace(p.Category)
	p.Description = strings.TrimSpace(p.Description)
	p.Address = strings.TrimSpace(p.Address)
	switch {
	case p.Name == "":
		writeError(w, http.StatusBadRequest, "name is required")
	case p.City == "":
		writeError(w, http.StatusBadRequest, "city is required")
	case p.Rating < 0 || p.Rating > 5:
		writeError(w, http.StatusBadRequest, "rating must be between 0 and 5")
	default:
		return p, true
	}
	return p, false
}

// respond writes v with status on success, 404 on errNotFound, else 500.
func respond(w http.ResponseWriter, status int, v any, err error, msg string) {
	switch {
	case errors.Is(err, errNotFound):
		writeError(w, http.StatusNotFound, "place not found")
	case err != nil:
		log.Printf("%s: %v", msg, err)
		writeError(w, http.StatusInternalServerError, msg)
	default:
		writeJSON(w, status, v)
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

func methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}
