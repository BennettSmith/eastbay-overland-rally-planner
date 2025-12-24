-- File: 01_enums.sql

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'trip_status') THEN
    CREATE TYPE trip_status AS ENUM ('DRAFT', 'PUBLISHED', 'CANCELED');
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'draft_visibility') THEN
    CREATE TYPE draft_visibility AS ENUM ('PRIVATE', 'PUBLIC');
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'rsvp_response') THEN
    CREATE TYPE rsvp_response AS ENUM ('YES', 'NO', 'UNSET');
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'artifact_type') THEN
    CREATE TYPE artifact_type AS ENUM ('GPX', 'SCHEDULE', 'DOCUMENT', 'OTHER');
  END IF;
END $$;
