# East Bay Overland — Local Dev Helpers
#
# Prereqs:
#   - Docker Desktop (or compatible)
#   - psql client (optional; also available via docker exec)
#
# Usage:
#   make up            # start postgres
#   make db-create     # create database (if not already created)
#   make db-migrate    # apply DDL (schema)
#   make db-seed       # seed sample data
#   make db-reset      # drop/recreate/apply/seed (destructive)
#
# Environment:
#   - You can override defaults by creating a .env file (see .env.example)
#   - Or pass variables inline: make db-reset POSTGRES_DB=ebo_dev

SHELL := /bin/bash

POSTGRES_USER ?= ebo
POSTGRES_PASSWORD ?= ebo_password
POSTGRES_DB ?= ebo_dev
POSTGRES_PORT ?= 5432

COMPOSE ?= docker compose
SERVICE ?= postgres

# Connection string for local host access (psql on host)
DATABASE_URL ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

# Path to the DDL folder (expects the DDL zip extracted next to this Makefile, or override)
DDL_DIR ?= ./ddl

PSQL ?= psql
PSQL_FLAGS ?= -v ON_ERROR_STOP=1

.PHONY: help
help:
	@echo "Targets:"
	@echo "  up             Start postgres container"
	@echo "  down           Stop postgres container"
	@echo "  logs           Tail postgres logs"
	@echo "  psql           Open interactive psql shell (inside container)"
	@echo "  db-create      Create database (if missing)"
	@echo "  db-drop        Drop database (destructive)"
	@echo "  db-migrate     Apply schema DDL (00..05)"
	@echo "  db-seed        Apply dev seed (06)"
	@echo "  db-reset       Drop + create + migrate + seed (destructive)"
	@echo ""
	@echo "Vars (override like: make up POSTGRES_PORT=5433):"
	@echo "  POSTGRES_USER POSTGRES_PASSWORD POSTGRES_DB POSTGRES_PORT DDL_DIR DATABASE_URL"

.PHONY: up
up:
	$(COMPOSE) up -d

.PHONY: down
down:
	$(COMPOSE) down

.PHONY: logs
logs:
	$(COMPOSE) logs -f $(SERVICE)

.PHONY: psql
psql:
	$(COMPOSE) exec -it $(SERVICE) psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

# --- Database lifecycle helpers (run via container to avoid host tooling dependencies) ---

.PHONY: db-create
db-create: up
	@$(COMPOSE) exec -T $(SERVICE) bash -lc '\
		psql -v ON_ERROR_STOP=1 -U "$(POSTGRES_USER)" -d postgres \
			-c "SELECT 1 FROM pg_database WHERE datname = '"'"'$(POSTGRES_DB)'"'"';" | grep -q 1 \
		|| psql -v ON_ERROR_STOP=1 -U "$(POSTGRES_USER)" -d postgres \
			-c "CREATE DATABASE \"$(POSTGRES_DB)\";" \
	'

.PHONY: db-drop
db-drop: up
	@$(COMPOSE) exec -T $(SERVICE) bash -lc '\
		psql -v ON_ERROR_STOP=1 -U "$(POSTGRES_USER)" -d postgres \
			-c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '"'"'$(POSTGRES_DB)'"'"' AND pid <> pg_backend_pid();" \
		&& psql -v ON_ERROR_STOP=1 -U "$(POSTGRES_USER)" -d postgres \
			-c "DROP DATABASE IF EXISTS \"$(POSTGRES_DB)\";" \
	'

.PHONY: db-migrate
db-migrate: db-create
	@test -d "$(DDL_DIR)" || (echo "DDL_DIR not found: $(DDL_DIR). Put DDL files in ./ddl or set DDL_DIR."; exit 1)
	@$(COMPOSE) exec -T $(SERVICE) bash -lc '\
		psql $(PSQL_FLAGS) -U "$(POSTGRES_USER)" -d "$(POSTGRES_DB)" -f /ddl/00_extensions.sql && \
		psql $(PSQL_FLAGS) -U "$(POSTGRES_USER)" -d "$(POSTGRES_DB)" -f /ddl/01_enums.sql && \
		psql $(PSQL_FLAGS) -U "$(POSTGRES_USER)" -d "$(POSTGRES_DB)" -f /ddl/02_tables.sql && \
		psql $(PSQL_FLAGS) -U "$(POSTGRES_USER)" -d "$(POSTGRES_DB)" -f /ddl/03_indexes.sql && \
		psql $(PSQL_FLAGS) -U "$(POSTGRES_USER)" -d "$(POSTGRES_DB)" -f /ddl/04_triggers.sql && \
		psql $(PSQL_FLAGS) -U "$(POSTGRES_USER)" -d "$(POSTGRES_DB)" -f /ddl/05_views.sql \
	'

.PHONY: db-seed
db-seed: db-migrate
	@test -d "$(DDL_DIR)" || (echo "DDL_DIR not found: $(DDL_DIR)."; exit 1)
	@$(COMPOSE) exec -T $(SERVICE) bash -lc '\
		psql $(PSQL_FLAGS) -U "$(POSTGRES_USER)" -d "$(POSTGRES_DB)" -f /ddl/06_seed_dev_optional.sql \
	'

.PHONY: db-reset
db-reset: db-drop db-create db-migrate db-seed
	@echo "Reset complete: $(POSTGRES_DB) on localhost:$(POSTGRES_PORT)"
