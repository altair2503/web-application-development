# City Insights API

Base URL: `http://localhost:8080`. All bodies are JSON; errors are `{"error": "..."}`.

## Place object
| field | type | notes |
|---|---|---|
| id | int | server-generated |
| name | string | required |
| city | string | required |
| category | string | optional (park, museum, viewpoint, cafe…) |
| description | string | optional |
| address | string | optional |
| rating | number | 0–5, default 0 |
| created_at | string | RFC 3339, server-generated |

## Endpoints
| method | path | success | errors |
|---|---|---|---|
| GET | `/api/places?city=&category=` | 200 array, sorted by rating desc (filters case-insensitive, optional) | 500 |
| GET | `/api/places/{id}` | 200 place | 400 bad id, 404 |
| POST | `/api/places` | 201 created place | 400 invalid JSON / missing name or city / rating outside 0–5 |
| PUT | `/api/places/{id}` | 200 updated place (full replace) | 400 (id or body), 404 |
| DELETE | `/api/places/{id}` | 204 no body | 400 bad id, 404 |
| GET | `/health` | 200 `{"status":"ok"}` | |
| GET | `/hello` | 200 text, shows `APP_NAME` | |

Any other method on `/api/places*` returns 405.

## Example
```bash
curl -X POST localhost:8080/api/places -H 'Content-Type: application/json' \
  -d '{"name":"Medeu","city":"Almaty","category":"sport","rating":4.7}'
```
```json
{"id":6,"name":"Medeu","city":"Almaty","category":"sport","description":"","address":"","rating":4.7,"created_at":"2026-10-31T10:00:00Z"}
```
