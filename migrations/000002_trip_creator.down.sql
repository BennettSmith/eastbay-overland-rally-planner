-- 000002_trip_creator.down.sql

-- Drop triggers first.
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_trips_add_creator_organizer') THEN
    DROP TRIGGER trg_trips_add_creator_organizer ON trips;
  END IF;
END $$;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_trips_prevent_creator_change') THEN
    DROP TRIGGER trg_trips_prevent_creator_change ON trips;
  END IF;
END $$;

-- Drop functions.
DROP FUNCTION IF EXISTS trips_add_creator_as_organizer();
DROP FUNCTION IF EXISTS prevent_trip_creator_change();

-- Drop FK + column.
ALTER TABLE trips
  DROP CONSTRAINT IF EXISTS trips_created_by_member_fk;

ALTER TABLE trips
  DROP COLUMN IF EXISTS created_by_member_id;


