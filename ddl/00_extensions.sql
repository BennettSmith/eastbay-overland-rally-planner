-- East Bay Overland — Trip Planning DB Schema (PostgreSQL)
-- File: 00_extensions.sql
--
-- Usage:
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f 00_extensions.sql
--
-- Notes:
-- - Uses pgcrypto for gen_random_uuid().
-- - Uses citext for case-insensitive email/search convenience.

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;
