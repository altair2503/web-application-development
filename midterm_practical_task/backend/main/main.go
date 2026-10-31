package main

import (
	"context"
	"encoding/json"
	"errors"
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

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Each route is registered with and without trailing slash.
	route := func(pattern string, h http.HandlerFunc) {
		method, path, _ := strings.Cut(pattern, " ")
		mux.HandleFunc(method+" "+path, h)
		mux.HandleFunc(method+" "+path+"/", h)
	}
	route("GET /api/criminals", listCriminalsHandler(pool))
	route("GET /api/criminals/{id}", getCriminalHandler(pool))
	route("GET /api/hunts", listHuntsHandler(pool))
	route("POST /api/hunts", createHuntHandler(pool))
	route("GET /api/hunts/{id}", getHuntHandler(pool))
	route("PATCH /api/hunts/{id}", patchHuntHandler(pool))
	route("DELETE /api/hunts/{id}", deleteHuntHandler(pool))

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func reqCtx(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 10*time.Second)
}

func listCriminalsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		var f criminalFilter
		var ok bool
		if f.MinReward, ok = queryFloat(w, q.Get("min_reward"), "min_reward"); !ok {
			return
		}
		if f.MaxReward, ok = queryFloat(w, q.Get("max_reward"), "max_reward"); !ok {
			return
		}
		if f.MinReward != nil && f.MaxReward != nil && *f.MinReward > *f.MaxReward {
			writeError(w, http.StatusBadRequest, "min_reward must not exceed max_reward")
			return
		}
		if f.Armed, ok = queryBool(w, q.Get("armed"), "armed"); !ok {
			return
		}
		if f.HasCrew, ok = queryBool(w, q.Get("has_crew"), "has_crew"); !ok {
			return
		}
		if f.EscapeMaster, ok = queryBool(w, q.Get("escape_master"), "escape_master"); !ok {
			return
		}
		f.Sort = q.Get("sort")
		if f.Sort != "" && f.Sort != "reward" && f.Sort != "-reward" {
			writeError(w, http.StatusBadRequest, "sort must be 'reward' or '-reward'")
			return
		}
		ctx, cancel := reqCtx(r)
		defer cancel()
		list, err := listCriminals(ctx, pool, f)
		respond(w, http.StatusOK, list, err)
	}
}

func getCriminalHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		ctx, cancel := reqCtx(r)
		defer cancel()
		c, err := getCriminal(ctx, pool, id)
		respond(w, http.StatusOK, c, err)
	}
}

func listHuntsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var cid *int
		if s := r.URL.Query().Get("criminal_id"); s != "" {
			v, err := strconv.Atoi(s)
			if err != nil || v < 1 {
				writeError(w, http.StatusBadRequest, "criminal_id must be a positive integer")
				return
			}
			cid = &v
		}
		ctx, cancel := reqCtx(r)
		defer cancel()
		list, err := listHunts(ctx, pool, cid)
		respond(w, http.StatusOK, list, err)
	}
}

func getHuntHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		ctx, cancel := reqCtx(r)
		defer cancel()
		h, err := getHunt(ctx, pool, id)
		respond(w, http.StatusOK, h, err)
	}
}

func createHuntHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// created_at (and id) sent by the client are ignored: unknown fields are not decoded.
		var in struct {
			CriminalID *int    `json:"criminal_id"`
			Success    *bool   `json:"success"`
			Notes      *string `json:"notes"`
		}
		if !decodeBody(w, r, &in, false) {
			return
		}
		switch {
		case in.CriminalID == nil:
			writeError(w, http.StatusBadRequest, "criminal_id is required")
			return
		case *in.CriminalID < 1:
			writeError(w, http.StatusBadRequest, "criminal_id must be a positive integer")
			return
		case in.Success == nil:
			writeError(w, http.StatusBadRequest, "success is required and must be boolean")
			return
		}
		notes := ""
		if in.Notes != nil {
			notes = strings.TrimSpace(*in.Notes)
		}
		ctx, cancel := reqCtx(r)
		defer cancel()
		h, err := createHunt(ctx, pool, *in.CriminalID, *in.Success, notes)
		respond(w, http.StatusCreated, h, err)
	}
}

func patchHuntHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		var in struct {
			Success *bool   `json:"success"`
			Notes   *string `json:"notes"`
		}
		// strict: only success and notes may be updated
		if !decodeBody(w, r, &in, true) {
			return
		}
		if in.Success == nil && in.Notes == nil {
			writeError(w, http.StatusBadRequest, "provide success and/or notes")
			return
		}
		if in.Notes != nil {
			t := strings.TrimSpace(*in.Notes)
			in.Notes = &t
		}
		ctx, cancel := reqCtx(r)
		defer cancel()
		h, err := updateHunt(ctx, pool, id, in.Success, in.Notes)
		respond(w, http.StatusOK, h, err)
	}
}

func deleteHuntHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		ctx, cancel := reqCtx(r)
		defer cancel()
		if err := deleteHunt(ctx, pool, id); err != nil {
			respond(w, 0, nil, err)
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

func queryFloat(w http.ResponseWriter, s, name string) (*float64, bool) {
	if s == "" {
		return nil, true
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v < 0 {
		writeError(w, http.StatusBadRequest, name+" must be a non-negative number")
		return nil, false
	}
	return &v, true
}

func queryBool(w http.ResponseWriter, s, name string) (*bool, bool) {
	if s == "" {
		return nil, true
	}
	switch strings.ToLower(s) {
	case "true", "1":
		v := true
		return &v, true
	case "false", "0":
		v := false
		return &v, true
	}
	writeError(w, http.StatusBadRequest, name+" must be true or false")
	return nil, false
}

func decodeBody(w http.ResponseWriter, r *http.Request, dst any, strict bool) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	if strict {
		dec.DisallowUnknownFields()
	}
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return false
	}
	return true
}

// respond writes v on success; maps errNotFound -> 404, errNoCriminal -> 400, otherwise 500.
func respond(w http.ResponseWriter, status int, v any, err error) {
	switch {
	case errors.Is(err, errNotFound):
		writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, errNoCriminal):
		writeError(w, http.StatusBadRequest, "criminal_id does not reference an existing criminal")
	case err != nil:
		log.Printf("error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
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
