# East Bay Overland — Local Dev

This repo uses:

- **Postgres via Docker Compose** (`db` service)
- **golang-migrate** via the `migrate/migrate` image (`migrate` service)
- A Go API (`api` service) behind Caddy locally (`caddy` service)

## Diagrams

- **Domain model (v1)**: `docs/diagrams/domain-model.md`
- **Database schema (v1)**: `docs/diagrams/database-schema.md`

## Local dev (DB + API)

Bring up the database and API (and the local proxy):

```bash
docker compose up -d --build db api caddy
```

Or, use the repo `Makefile` helpers (recommended):

```bash
make up
make db-migrate
```

### Developer flows (Docker Desktop)

Common compose-based workflows:

- **Start everything (db + migrate + api + caddy)**:

```bash
make up
```

- **Local dev auth**: this compose setup runs the API with `AUTH_MODE=dev`, which means:
  - Requests authenticate via `X-Debug-Subject: <some-subject>`
  - The API still enforces “member must be provisioned” for many endpoints, so you’ll typically create a member first.

- **If you get “port 5432 is already allocated”** (another Postgres is already using it):

```bash
make up POSTGRES_PORT=5433
```

- **Check status / ports**:

```bash
make ps
```

- **Tail logs**:

```bash
make logs-api   # API only
make logs-all   # everything
```

- **Rebuild only the API image (compose)**:

```bash
make rebuild-api
make up
```

- **Reset the Postgres volume (destructive; wipes all local data)**:

```bash
make reset-volumes
make up
```

Optional: build/run the API image without compose (useful for validating the container itself):

```bash
make image
make image-run DATABASE_URL='postgres://eb:eb@host.docker.internal:5432/eastbay?sslmode=disable'
```

### Quick curl checks

- **API health (through Caddy)**:

```bash
curl -i http://localhost:8081/healthz
```

- **Create a member (first-time setup for a subject)**:

```bash
curl -sS -X POST http://localhost:8081/members \
  -H 'Content-Type: application/json' \
  -H 'X-Debug-Subject: dev|alice' \
  -d '{"displayName":"Alice","email":"alice@example.com"}'
```

- **List members**:

```bash
curl -sS http://localhost:8081/members -H 'X-Debug-Subject: dev|alice'
```

Database connection string (from host):

```bash
postgres://eb:eb@localhost:5432/eastbay?sslmode=disable
```

## Fast dev loop (spec-first)

OpenAPI is the source of truth (`openapi.yaml`). Regenerate server glue, format, and run tests:

```bash
make gen-openapi
make fmt
make test
make cover
```

If you want an HTML coverage report:

```bash
make cover-html
```

## Environment variables

- **Auth (required)**:
  - `JWT_ISSUER`
  - `JWT_AUDIENCE`
  - `JWT_JWKS_URL`
- **Storage backend**:
  - `STORAGE_BACKEND`: `memory` (default) or `postgres`
  - `DATABASE_URL`: required when `STORAGE_BACKEND=postgres`
- **Postgres contract tests (optional)**:
  - `PG_DSN`: if set, Postgres adapter contract tests will run (they reset the `public` schema; use a disposable database).

## Run migrations

Apply migrations (defaults to `up`):

```bash
docker compose run --rm migrate
```

Reset schema (destructive):

```bash
docker compose run --rm migrate down -all
docker compose run --rm migrate
```

## Optional: dev seed (NOT part of migrations)

Seed data lives in `db/seed/seed_dev_optional.sql` and is intentionally **not** included in `migrations/`.

If you want sample data in a local dev database, you can run it against the running container:

```bash
docker compose exec -T db psql -U eb -d eastbay -v ON_ERROR_STOP=1 < db/seed/seed_dev_optional.sql
```

Or, run it from your host with `DATABASE_URL`:

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f db/seed/seed_dev_optional.sql
```

Example host DB URL (when using Docker Compose locally):

```bash
postgres://eb:eb@localhost:5432/eastbay?sslmode=disable
```
