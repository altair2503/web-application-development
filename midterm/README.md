# City Insights — midterm project (Part 1)

REST API for beautiful places in a city. Go + PostgreSQL 16, run with Docker Compose.
Full endpoint reference: [API.md](API.md). Postman: [postman/](postman/).

## Architecture
```
client ──:8080──> [ api (Go, custom Dockerfile) ] ──city-insights-net──> [ db (postgres:16-alpine) ]
                                                                              │
                                                              named volume city_insights_pg_data
```

## Run
```bash
cp .env.example .env      # edit values if needed
docker compose up --build
curl localhost:8080/api/places
```
The `places` table is created and seeded with 5 sample places on first start.

## Docker pieces
- **Dockerfile** (`backend/`): multi-stage — `golang:1.23-alpine` builds a static binary, `alpine` runs it (small image).
- **Network** `city-insights-net`: services reach each other by name (`DB_HOST=db`); Postgres port 5432 is not published to the host.
- **Volume** `city_insights_pg_data`: DB files survive `docker compose down`; `docker compose down -v` wipes them.
- **Env vars** from `.env` (`POSTGRES_*`, `DB_HOST`, `DB_PORT`, `APP_PORT`, `APP_NAME`); `.env` is gitignored, `.env.example` is committed.
- **Healthcheck + `depends_on: service_healthy`**: API starts only once Postgres accepts connections (API also retries on connect).

## Endpoints
| method | path | description |
|---|---|---|
| GET | `/api/places?city=&category=` | list / filter |
| GET | `/api/places/{id}` | get one |
| POST | `/api/places` | create |
| PUT | `/api/places/{id}` | update |
| DELETE | `/api/places/{id}` | delete |
| GET | `/health`, `/hello` | health, env-var demo |

## Test
Import `postman/City-Insights.postman_collection.json` and `postman/City-Insights.postman_environment.json` into Postman, or:
```bash
npx newman run postman/City-Insights.postman_collection.json -e postman/City-Insights.postman_environment.json
```
