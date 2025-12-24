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

- Copy `.env.example` to `.env` and adjust values as needed.
- Auth variables (`JWT_ISSUER`, `JWT_AUDIENCE`, `JWT_JWKS_URL`) will become required once Milestone 1 is implemented.

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
