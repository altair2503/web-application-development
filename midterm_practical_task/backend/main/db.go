package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schemaSQL string

//go:embed seed.sql
var seedSQL string

var (
	errNotFound   = errors.New("not found")
	errNoCriminal = errors.New("criminal does not exist")
)

type Criminal struct {
	ID           int      `json:"id"`
	Name         *string  `json:"name"`
	Nickname     string   `json:"nickname"`
	Reward       float64  `json:"reward"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
	LocationNote *string  `json:"location_note"`
	Difficulty   int      `json:"difficulty"`
	Armed        bool     `json:"armed"`
	HasCrew      bool     `json:"has_crew"`
	EscapeMaster bool     `json:"escape_master"`
}

type HuntLog struct {
	ID         int       `json:"id"`
	CriminalID int       `json:"criminal_id"`
	Success    bool      `json:"success"`
	Notes      string    `json:"notes"`
	CreatedAt  time.Time `json:"created_at"`
}

const criminalCols = `id, name, nickname, reward::float8, latitude, longitude, location_note, difficulty, armed, has_crew, escape_master`
const huntCols = `id, criminal_id, success, notes, created_at`

func connectDB(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	const maxAttempts = 30
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		pool, err := pgxpool.New(ctx, databaseURL)
		if err == nil {
			if err = pool.Ping(ctx); err == nil {
				if _, err = pool.Exec(ctx, schemaSQL); err == nil {
					_, err = pool.Exec(ctx, seedSQL)
				}
			}
			if err == nil {
				return pool, nil
			}
			pool.Close()
		}
		lastErr = err
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("database unavailable after %d attempts: %w", maxAttempts, lastErr)
}

func scanCriminal(row pgx.Row) (Criminal, error) {
	var c Criminal
	err := row.Scan(&c.ID, &c.Name, &c.Nickname, &c.Reward, &c.Latitude, &c.Longitude,
		&c.LocationNote, &c.Difficulty, &c.Armed, &c.HasCrew, &c.EscapeMaster)
	if errors.Is(err, pgx.ErrNoRows) {
		err = errNotFound
	}
	return c, err
}

func scanHunt(row pgx.Row) (HuntLog, error) {
	var h HuntLog
	err := row.Scan(&h.ID, &h.CriminalID, &h.Success, &h.Notes, &h.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = errNotFound
	}
	return h, err
}

type criminalFilter struct {
	MinReward, MaxReward         *float64
	Armed, HasCrew, EscapeMaster *bool
	Sort                         string // "", "reward", "-reward"
}

// listCriminals builds the WHERE clause from only the filters provided, so all filters combine with AND.
func listCriminals(ctx context.Context, pool *pgxpool.Pool, f criminalFilter) ([]Criminal, error) {
	var where []string
	var args []any
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}
	if f.MinReward != nil {
		add("reward >= $%d", *f.MinReward)
	}
	if f.MaxReward != nil {
		add("reward <= $%d", *f.MaxReward)
	}
	if f.Armed != nil {
		add("armed = $%d", *f.Armed)
	}
	if f.HasCrew != nil {
		add("has_crew = $%d", *f.HasCrew)
	}
	if f.EscapeMaster != nil {
		add("escape_master = $%d", *f.EscapeMaster)
	}
	q := `SELECT ` + criminalCols + ` FROM wanted_criminals`
	if len(where) > 0 {
		q += ` WHERE ` + strings.Join(where, " AND ")
	}
	switch f.Sort {
	case "reward":
		q += ` ORDER BY reward ASC, id`
	case "-reward":
		q += ` ORDER BY reward DESC, id`
	default:
		q += ` ORDER BY id`
	}
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Criminal{}
	for rows.Next() {
		c, err := scanCriminal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func getCriminal(ctx context.Context, pool *pgxpool.Pool, id int) (Criminal, error) {
	return scanCriminal(pool.QueryRow(ctx, `SELECT `+criminalCols+` FROM wanted_criminals WHERE id=$1`, id))
}

func listHunts(ctx context.Context, pool *pgxpool.Pool, criminalID *int) ([]HuntLog, error) {
	rows, err := pool.Query(ctx,
		`SELECT `+huntCols+` FROM hunt_logs WHERE ($1::int IS NULL OR criminal_id = $1) ORDER BY id`, criminalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HuntLog{}
	for rows.Next() {
		h, err := scanHunt(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func getHunt(ctx context.Context, pool *pgxpool.Pool, id int) (HuntLog, error) {
	return scanHunt(pool.QueryRow(ctx, `SELECT `+huntCols+` FROM hunt_logs WHERE id=$1`, id))
}

func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func createHunt(ctx context.Context, pool *pgxpool.Pool, criminalID int, success bool, notes string) (HuntLog, error) {
	h, err := scanHunt(pool.QueryRow(ctx,
		`INSERT INTO hunt_logs (criminal_id, success, notes) VALUES ($1,$2,$3) RETURNING `+huntCols,
		criminalID, success, notes))
	if isFKViolation(err) {
		err = errNoCriminal
	}
	return h, err
}

// updateHunt changes only the provided fields (nil = keep current value).
func updateHunt(ctx context.Context, pool *pgxpool.Pool, id int, success *bool, notes *string) (HuntLog, error) {
	return scanHunt(pool.QueryRow(ctx,
		`UPDATE hunt_logs SET success = COALESCE($2, success), notes = COALESCE($3, notes)
		 WHERE id=$1 RETURNING `+huntCols, id, success, notes))
}

func deleteHunt(ctx context.Context, pool *pgxpool.Pool, id int) error {
	tag, err := pool.Exec(ctx, `DELETE FROM hunt_logs WHERE id=$1`, id)
	if err == nil && tag.RowsAffected() == 0 {
		err = errNotFound
	}
	return err
}
