# Space Bounty Hunters API

REST API for bounty hunters: database of wanted criminals + hunt journal.
Stack (same as my projects): **Go (net/http) + pgx + PostgreSQL 16 + Docker Compose**.

## Run
```bash
cp .env.example .env
docker compose up --build -d
curl "localhost:8080/api/criminals/?min_reward=5000&armed=true&sort=-reward"
```
## Database setup / import
Automatic on API start (`backend/main/schema.sql`, `seed.sql`, embedded in the binary):
tables `wanted_criminals`, `hunt_logs` are created and, if empty, filled with **7 criminals and 12 hunt logs**.
Reset data: `docker compose down -v`.

## Models
- **WantedCriminal**: id, name (nullable), nickname, reward, latitude/longitude (nullable) or location_note (text), difficulty (1–10), armed, has_crew, escape_master.
- **HuntLog**: id, criminal_id (FK → WantedCriminal, `ON DELETE CASCADE`), success, notes, created_at (server-generated).
- One criminal → many hunt logs.

## Endpoints
| method | path | notes |
|---|---|---|
| GET | `/api/criminals/` | `min_reward`, `max_reward`, `armed`, `has_crew`, `escape_master`, `sort=reward\|-reward` (all combinable) |
| GET | `/api/criminals/{id}/` | 200 / 404 |
| GET | `/api/hunts/` | `criminal_id` filter |
| GET | `/api/hunts/{id}/` | 200 / 404 |
| POST | `/api/hunts/` | body `criminal_id`, `success`, `notes`; 201; `created_at` from client is ignored |
| PATCH | `/api/hunts/{id}/` | only `success` and/or `notes`; other fields → 400 |
| DELETE | `/api/hunts/{id}/` | 204 / 404 |

Errors: `400` invalid id / JSON / params / unknown criminal_id, `404` not found. Body: `{"error": "..."}`.
Trailing slash is optional.

## Postman
Import `postman/Space-Bounty-Hunters.postman_collection.json` and `postman/Space-Bounty-Hunters.postman_environment.json` (`baseUrl=http://localhost:8080`), or:
```bash
npx newman run postman/Space-Bounty-Hunters.postman_collection.json -e postman/Space-Bounty-Hunters.postman_environment.json
```
Collection contains: listing/filtering/sorting, details, hunt CRUD (id is chained via a variable), invalid requests.
