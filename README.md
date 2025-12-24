# East Bay Overland — Local Postgres Dev Setup

This folder provides a **Docker Compose** Postgres instance and a **Makefile** with common tasks
(create/migrate/seed/reset).

## Quick start

1. Start Postgres

   ```bash
   make up
   ```

2. Create + apply schema + seed sample data

   ```bash
   make db-reset
   ```

3. Connect

   - Inside container:

     ```bash
     make psql
     ```

   - From host (if you have `psql` installed):

     ```bash
     psql "postgres://ebo:ebo_password@localhost:5432/ebo_dev?sslmode=disable"
     ```

## Notes

- The Makefile runs all `psql` commands **inside the container** to reduce host dependencies.
- DDL files are mounted read-only at `/ddl`.
