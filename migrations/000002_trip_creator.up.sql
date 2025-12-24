-- 000002_trip_creator.up.sql
--
-- Add an explicit creator for each trip to support "private draft is visible only to creator"
-- and to keep an audit trail independent of organizer membership changes.

-- 1) Add column (nullable for backfill), then FK.
ALTER TABLE trips
  ADD COLUMN IF NOT EXISTS created_by_member_id bigint NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'trips_created_by_member_fk'
  ) THEN
    ALTER TABLE trips
      ADD CONSTRAINT trips_created_by_member_fk
      FOREIGN KEY (created_by_member_id) REFERENCES members(id) ON DELETE RESTRICT;
  END IF;
END $$;

-- 2) Backfill from earliest organizer row (best-effort for existing data).
WITH first_org AS (
  SELECT DISTINCT ON (trip_id)
    trip_id,
    member_id
  FROM trip_organizers
  ORDER BY trip_id, added_at ASC, member_id ASC
)
UPDATE trips t
SET created_by_member_id = fo.member_id
FROM first_org fo
WHERE t.id = fo.trip_id
  AND t.created_by_member_id IS NULL;

-- Ensure we didn't leave any rows without a creator.
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM trips WHERE created_by_member_id IS NULL) THEN
    RAISE EXCEPTION 'Backfill failed: some trips have no created_by_member_id. Ensure each trip has at least one organizer before applying this migration.';
  END IF;
END $$;

ALTER TABLE trips
  ALTER COLUMN created_by_member_id SET NOT NULL;

-- 3) Prevent changing the creator once set (immutable audit field).
CREATE OR REPLACE FUNCTION prevent_trip_creator_change()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF (NEW.created_by_member_id IS DISTINCT FROM OLD.created_by_member_id) THEN
    RAISE EXCEPTION 'Trip creator cannot be changed (trip_id=%)', OLD.id
      USING ERRCODE = '23514';
  END IF;
  RETURN NEW;
END;
$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_trips_prevent_creator_change') THEN
    CREATE TRIGGER trg_trips_prevent_creator_change
    BEFORE UPDATE OF created_by_member_id ON trips
    FOR EACH ROW
    EXECUTE FUNCTION prevent_trip_creator_change();
  END IF;
END $$;

-- 4) Ensure creator is always an organizer on insert (Option A).
CREATE OR REPLACE FUNCTION trips_add_creator_as_organizer()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  INSERT INTO trip_organizers (trip_id, member_id)
  VALUES (NEW.id, NEW.created_by_member_id)
  ON CONFLICT DO NOTHING;
  RETURN NEW;
END;
$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_trips_add_creator_organizer') THEN
    CREATE TRIGGER trg_trips_add_creator_organizer
    AFTER INSERT ON trips
    FOR EACH ROW
    EXECUTE FUNCTION trips_add_creator_as_organizer();
  END IF;
END $$;


