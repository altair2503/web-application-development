package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const migrateSQL = `
CREATE TABLE IF NOT EXISTS quotes (
  id         SERIAL PRIMARY KEY,
  text       TEXT NOT NULL,
  author     TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`

type Quote struct {
	ID        int       `json:"id"`
	Text      string    `json:"text"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
}

func connectDB(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	const maxAttempts = 30
	const delay = 2 * time.Second

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		pool, err := pgxpool.New(ctx, databaseURL)
		if err != nil {
			lastErr = err
			time.Sleep(delay)
			continue
		}
		if err := pool.Ping(ctx); err != nil {
			pool.Close()
			lastErr = err
			time.Sleep(delay)
			continue
		}
		if _, err := pool.Exec(ctx, migrateSQL); err != nil {
			pool.Close()
			lastErr = err
			time.Sleep(delay)
			continue
		}
		return pool, nil
	}
	return nil, fmt.Errorf("database unavailable after %d attempts: %w", maxAttempts, lastErr)
}

func listQuotes(ctx context.Context, pool *pgxpool.Pool) ([]Quote, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, text, author, created_at FROM quotes ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quotes []Quote
	for rows.Next() {
		var q Quote
		if err := rows.Scan(&q.ID, &q.Text, &q.Author, &q.CreatedAt); err != nil {
			return nil, err
		}
		quotes = append(quotes, q)
	}
	return quotes, rows.Err()
}

func createQuote(ctx context.Context, pool *pgxpool.Pool, text, author string) (Quote, error) {
	var q Quote
	err := pool.QueryRow(ctx,
		`INSERT INTO quotes (text, author) VALUES ($1, $2)
		 RETURNING id, text, author, created_at`,
		text, author,
	).Scan(&q.ID, &q.Text, &q.Author, &q.CreatedAt)
	return q, err
}
