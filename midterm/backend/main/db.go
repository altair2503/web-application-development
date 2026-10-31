package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var errNotFound = errors.New("not found")

const migrateSQL = `
CREATE TABLE IF NOT EXISTS places (
  id          SERIAL PRIMARY KEY,
  name        TEXT NOT NULL,
  city        TEXT NOT NULL,
  category    TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  address     TEXT NOT NULL DEFAULT '',
  rating      NUMERIC(2,1) NOT NULL DEFAULT 0,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`

const seedSQL = `
INSERT INTO places (name, city, category, description, address, rating)
SELECT * FROM (VALUES
  ('Charyn Canyon', 'Almaty', 'nature', 'Red sandstone canyon, the Valley of Castles.', 'Charyn National Park', 4.9::numeric),
  ('Kok Tobe', 'Almaty', 'viewpoint', 'Hilltop with cable car and city panorama.', 'Kok Tobe Hill', 4.6::numeric),
  ('Big Almaty Lake', 'Almaty', 'nature', 'Turquoise alpine lake in the Trans-Ili Alatau.', 'Ile-Alatau National Park', 4.8::numeric),
  ('Baiterek Tower', 'Astana', 'landmark', 'Iconic observation tower with golden sphere.', 'Nurzhol Blvd', 4.7::numeric),
  ('Khan Shatyr', 'Astana', 'entertainment', 'Giant tent-shaped shopping and leisure centre.', '37 Turan Ave', 4.5::numeric)
) AS v(name, city, category, description, address, rating)
WHERE NOT EXISTS (SELECT 1 FROM places);
`

const placeCols = `id, name, city, category, description, address, rating::float8, created_at`

type Place struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	City        string    `json:"city"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Address     string    `json:"address"`
	Rating      float64   `json:"rating"`
	CreatedAt   time.Time `json:"created_at"`
}

func connectDB(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	const maxAttempts = 30
	const delay = 2 * time.Second

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		pool, err := pgxpool.New(ctx, databaseURL)
		if err == nil {
			if err = pool.Ping(ctx); err == nil {
				if _, err = pool.Exec(ctx, migrateSQL); err == nil {
					_, err = pool.Exec(ctx, seedSQL)
				}
			}
			if err == nil {
				return pool, nil
			}
			pool.Close()
		}
		lastErr = err
		time.Sleep(delay)
	}
	return nil, fmt.Errorf("database unavailable after %d attempts: %w", maxAttempts, lastErr)
}

func scanPlace(row pgx.Row) (Place, error) {
	var p Place
	err := row.Scan(&p.ID, &p.Name, &p.City, &p.Category, &p.Description, &p.Address, &p.Rating, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = errNotFound
	}
	return p, err
}

// listPlaces filters by city/category (case-insensitive) when non-empty.
func listPlaces(ctx context.Context, pool *pgxpool.Pool, city, category string) ([]Place, error) {
	rows, err := pool.Query(ctx,
		`SELECT `+placeCols+` FROM places
		 WHERE ($1 = '' OR LOWER(city) = LOWER($1))
		   AND ($2 = '' OR LOWER(category) = LOWER($2))
		 ORDER BY rating DESC, id`, city, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	places := []Place{}
	for rows.Next() {
		p, err := scanPlace(rows)
		if err != nil {
			return nil, err
		}
		places = append(places, p)
	}
	return places, rows.Err()
}

func getPlace(ctx context.Context, pool *pgxpool.Pool, id int) (Place, error) {
	return scanPlace(pool.QueryRow(ctx, `SELECT `+placeCols+` FROM places WHERE id = $1`, id))
}

func createPlace(ctx context.Context, pool *pgxpool.Pool, p Place) (Place, error) {
	return scanPlace(pool.QueryRow(ctx,
		`INSERT INTO places (name, city, category, description, address, rating)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING `+placeCols,
		p.Name, p.City, p.Category, p.Description, p.Address, p.Rating))
}

func updatePlace(ctx context.Context, pool *pgxpool.Pool, id int, p Place) (Place, error) {
	return scanPlace(pool.QueryRow(ctx,
		`UPDATE places SET name=$2, city=$3, category=$4, description=$5, address=$6, rating=$7
		 WHERE id=$1 RETURNING `+placeCols,
		id, p.Name, p.City, p.Category, p.Description, p.Address, p.Rating))
}

func deletePlace(ctx context.Context, pool *pgxpool.Pool, id int) error {
	tag, err := pool.Exec(ctx, `DELETE FROM places WHERE id = $1`, id)
	if err == nil && tag.RowsAffected() == 0 {
		err = errNotFound
	}
	return err
}
