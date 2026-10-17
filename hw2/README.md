# Homework 2 — Multi-Container Quotes API

Go HTTP API backed by PostgreSQL, orchestrated with Docker Compose.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) with Docker Compose v2

## Setup

From this directory (`hw2/`):

```bash
cp .env.example .env
```

Edit `.env` if you want different credentials locally. Do not commit `.env`.

## Run

```bash
docker compose up --build
```

Services:

| Service | Name | Description |
|---------|------|-------------|
| API | `api` | Quotes REST API on port `8080` (host port from `APP_PORT`) |
| Database | `db` | PostgreSQL 16, data in named volume `hw2_pg_data` |

## API

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/hello` | Hello message using `APP_NAME` |
| `GET` | `/api/quotes` | List all quotes (newest first) |
| `POST` | `/api/quotes` | Create a quote (`text` required, `author` optional) |

### Examples

```bash
curl http://localhost:8080/health
curl http://localhost:8080/hello

curl -X POST http://localhost:8080/api/quotes \
  -H 'Content-Type: application/json' \
  -d '{"text":"Stay hungry, stay foolish.","author":"Steve Jobs"}'

curl http://localhost:8080/api/quotes
```

## Data persistence

Database files are stored in the Docker named volume `hw2_pg_data`. After `docker compose down`, start again with `docker compose up` — quotes remain.

To remove containers **and** delete stored data:

```bash
docker compose down -v
```

## Stop

```bash
docker compose down
```
