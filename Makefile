# East Bay Overland — Local Dev Helpers
#
# Prereqs:
#   - Docker Desktop (or compatible)
#   - psql client (optional; also available via docker exec)
#
# Usage:
#   make up            # start the full docker-compose stack (db+migrate+api+caddy)
#   make db-create     # create database (if not already created)
#   make db-migrate    # apply migrations (schema)
#   make db-seed       # seed sample data
#   make db-reset      # drop/recreate/apply/seed (destructive)
#
# Environment:
#   - You can override defaults by creating a .env file (see .env.example)
#   - Or pass variables inline: make db-reset POSTGRES_DB=ebo_dev

SHELL := /bin/bash

# Keep these defaults aligned with docker-compose.yml (so "make up" works out of the box).
POSTGRES_USER ?= eb
POSTGRES_PASSWORD ?= eb
POSTGRES_DB ?= eastbay
POSTGRES_PORT ?= 5432

COMPOSE ?= docker compose
DB_SERVICE ?= db
API_SERVICE ?= api
PROXY_SERVICE ?= caddy

# For building/running the API image outside compose (Docker Desktop-friendly).
API_IMAGE ?= ebo-api:local

# Connection string for local host access (psql on host)
DATABASE_URL ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

# Export these so `docker compose` picks them up for variable interpolation.
export POSTGRES_USER
export POSTGRES_PASSWORD
export POSTGRES_DB
export POSTGRES_PORT

# Dev-only seed script (must NOT be included in migrations)
SEED_FILE ?= ./db/seed/seed_dev_optional.sql

PSQL ?= psql
PSQL_FLAGS ?= -v ON_ERROR_STOP=1

.PHONY: help
help:
	@echo "Targets:"
	@echo "  up             Start docker-compose stack (db + migrate + api + caddy)"
	@echo "  down           Stop postgres container"
	@echo "  logs           Tail postgres logs"
	@echo "  ps             Show compose services"
	@echo "  logs-api       Tail API logs"
	@echo "  logs-all       Tail logs for all services"
	@echo "  rebuild-api    Rebuild the API container image (compose)"
	@echo "  reset-volumes  Stop stack and delete volumes (destructive; wipes dbdata)"
	@echo "  psql           Open interactive psql shell (inside container)"
	@echo "  db-create      Create database (if missing)"
	@echo "  db-drop        Drop database (destructive)"
	@echo "  db-migrate     Apply schema migrations (golang-migrate)"
	@echo "  db-seed        Apply dev seed (optional)"
	@echo "  db-reset       Drop + create + migrate + seed (destructive)"
	@echo ""
	@echo "  image          Build the API Docker image (no compose): $(API_IMAGE)"
	@echo "  image-run      Run the API Docker image (no compose; requires DATABASE_URL reachable)"
	@echo ""
	@echo "  fmt            Run gofmt on all .go files"
	@echo "  test           Run Go unit tests (./...)"
	@echo "  cover          Run tests with coverage (writes coverage.out; prints summary)"
	@echo "  cover-html     Generate coverage.html from coverage.out"
	@echo ""
	@echo "Vars (override like: make up POSTGRES_PORT=5433):"
	@echo "  POSTGRES_USER POSTGRES_PASSWORD POSTGRES_DB POSTGRES_PORT SEED_FILE DATABASE_URL"
	@echo "  COMPOSE DB_SERVICE API_SERVICE PROXY_SERVICE API_IMAGE"

.PHONY: up
up:
	$(COMPOSE) up -d --build --remove-orphans

.PHONY: down
down:
	$(COMPOSE) down

.PHONY: logs
logs:
	$(COMPOSE) logs -f $(DB_SERVICE)

.PHONY: ps
ps:
	$(COMPOSE) ps

.PHONY: logs-api
logs-api:
	$(COMPOSE) logs -f $(API_SERVICE)

.PHONY: logs-all
logs-all:
	$(COMPOSE) logs -f

.PHONY: rebuild-api
rebuild-api:
	$(COMPOSE) build --no-cache $(API_SERVICE)

.PHONY: reset-volumes
reset-volumes:
	$(COMPOSE) down -v --remove-orphans

.PHONY: migrate
migrate:
	$(COMPOSE) run --rm migrate

.PHONY: psql
psql:
	$(COMPOSE) exec -it $(DB_SERVICE) psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

# --- Database lifecycle helpers (run via container to avoid host tooling dependencies) ---

.PHONY: db-create
db-create: up
	@$(COMPOSE) exec -T $(DB_SERVICE) bash -lc '\
		psql -v ON_ERROR_STOP=1 -U "$(POSTGRES_USER)" -d postgres \
			-c "SELECT 1 FROM pg_database WHERE datname = '"'"'$(POSTGRES_DB)'"'"';" | grep -q 1 \
		|| psql -v ON_ERROR_STOP=1 -U "$(POSTGRES_USER)" -d postgres \
			-c "CREATE DATABASE \"$(POSTGRES_DB)\";" \
	'

.PHONY: db-drop
db-drop: up
	@$(COMPOSE) exec -T $(DB_SERVICE) bash -lc '\
		psql -v ON_ERROR_STOP=1 -U "$(POSTGRES_USER)" -d postgres \
			-c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '"'"'$(POSTGRES_DB)'"'"' AND pid <> pg_backend_pid();" \
		&& psql -v ON_ERROR_STOP=1 -U "$(POSTGRES_USER)" -d postgres \
			-c "DROP DATABASE IF EXISTS \"$(POSTGRES_DB)\";" \
	'

.PHONY: db-migrate
db-migrate: db-create
	@$(COMPOSE) run --rm migrate

.PHONY: db-seed
db-seed: db-migrate
	@test -f "$(SEED_FILE)" || (echo "SEED_FILE not found: $(SEED_FILE)."; exit 1)
	@$(COMPOSE) exec -T $(DB_SERVICE) bash -lc '\
		psql $(PSQL_FLAGS) -U "$(POSTGRES_USER)" -d "$(POSTGRES_DB)" \
	' < "$(SEED_FILE)"

.PHONY: db-reset
db-reset: db-drop db-create db-migrate db-seed
	@echo "Reset complete: $(POSTGRES_DB) on localhost:$(POSTGRES_PORT)"

# --- Go / OpenAPI helpers ---
#
# Generates Go server stubs + types from ./openapi.yaml.
# - Uses oapi-codegen (Go-native generator)
# - Targets net/http + chi (but keeps generated code isolated in a package you can adapt from)
#
# Note: pin the oapi-codegen version once you settle on it; `@latest` is convenient early on.
.PHONY: gen-openapi
gen-openapi:
	@go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.5.1 \
		-config oapi-codegen.yaml \
		openapi.yaml

# --- Go quality-of-life targets (Milestone 0) ---

GO ?= go
PKGS ?= ./...

COVERPROFILE ?= coverage.out
COVERHTML ?= coverage.html

.PHONY: fmt
fmt:
	@echo "gofmt..."
	@gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

.PHONY: test
test:
	@$(GO) test $(PKGS)

.PHONY: cover
cover:
	@$(GO) test -coverprofile=$(COVERPROFILE) $(PKGS)
	@$(GO) tool cover -func=$(COVERPROFILE)

.PHONY: cover-html
cover-html: cover
	@$(GO) tool cover -html=$(COVERPROFILE) -o $(COVERHTML)
	@echo "Wrote $(COVERHTML)"

# --- Docker image helpers (outside compose) ---

.PHONY: image
image:
	docker build -t $(API_IMAGE) .

.PHONY: image-run
image-run:
	docker run --rm -p 8080:8080 \
		-e PORT=8080 \
		-e STORAGE_BACKEND=postgres \
		-e DATABASE_URL='$(DATABASE_URL)' \
		$(API_IMAGE)

