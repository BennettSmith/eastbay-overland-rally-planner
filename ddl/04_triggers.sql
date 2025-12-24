-- File: 04_triggers.sql

-- =========================================================================
-- updated_at helpers
-- =========================================================================
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  NEW.updated_at := now();
  RETURN NEW;
END;
$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_members_set_updated_at') THEN
    CREATE TRIGGER trg_members_set_updated_at
    BEFORE UPDATE ON members
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_trips_set_updated_at') THEN
    CREATE TRIGGER trg_trips_set_updated_at
    BEFORE UPDATE ON trips
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_trip_artifacts_set_updated_at') THEN
    CREATE TRIGGER trg_trip_artifacts_set_updated_at
    BEFORE UPDATE ON trip_artifacts
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
  END IF;
END $$;

-- Member vehicle profile updated_at
CREATE OR REPLACE FUNCTION set_vehicle_profile_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  NEW.updated_at := now();
  RETURN NEW;
END;
$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_member_vehicle_profiles_set_updated_at') THEN
    CREATE TRIGGER trg_member_vehicle_profiles_set_updated_at
    BEFORE UPDATE ON member_vehicle_profiles
    FOR EACH ROW
    EXECUTE FUNCTION set_vehicle_profile_updated_at();
  END IF;
END $$;

-- RSVP updated_at
CREATE OR REPLACE FUNCTION set_rsvp_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  NEW.updated_at := now();
  RETURN NEW;
END;
$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_trip_rsvps_set_updated_at') THEN
    CREATE TRIGGER trg_trip_rsvps_set_updated_at
    BEFORE UPDATE ON trip_rsvps
    FOR EACH ROW
    EXECUTE FUNCTION set_rsvp_updated_at();
  END IF;
END $$;

-- =========================================================================
-- Organizer invariants: at least one organizer must always exist
-- =========================================================================
CREATE OR REPLACE FUNCTION prevent_removing_last_organizer()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  remaining_count integer;
BEGIN
  SELECT count(*) INTO remaining_count
  FROM trip_organizers
  WHERE trip_id = OLD.trip_id
    AND member_id <> OLD.member_id;

  IF remaining_count = 0 THEN
    RAISE EXCEPTION 'Cannot remove last organizer for trip %', OLD.trip_id
      USING ERRCODE = '23514'; -- check_violation -> map to 409 in app
  END IF;

  RETURN OLD;
END;
$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_trip_organizers_prevent_last_delete') THEN
    CREATE TRIGGER trg_trip_organizers_prevent_last_delete
    BEFORE DELETE ON trip_organizers
    FOR EACH ROW
    EXECUTE FUNCTION prevent_removing_last_organizer();
  END IF;
END $$;

-- =========================================================================
-- Publish/Cancel transitions: set published_at / canceled_at and enforce publish-required fields
-- =========================================================================
CREATE OR REPLACE FUNCTION trips_enforce_transitions()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  organizer_count integer;
BEGIN
  -- If status is changing to PUBLISHED, enforce required-at-publish fields (v1).
  IF (OLD.status <> 'PUBLISHED' AND NEW.status = 'PUBLISHED') THEN
    -- Must come from DRAFT
    IF OLD.status <> 'DRAFT' THEN
      RAISE EXCEPTION 'Trip can only be published from DRAFT (was %)', OLD.status
        USING ERRCODE = '23514';
    END IF;

    -- Required fields
    IF NEW.name IS NULL OR btrim(NEW.name) = '' THEN
      RAISE EXCEPTION 'Trip name is required to publish' USING ERRCODE = '23514';
    END IF;
    IF NEW.description IS NULL OR btrim(NEW.description) = '' THEN
      RAISE EXCEPTION 'Trip description is required to publish' USING ERRCODE = '23514';
    END IF;
    IF NEW.start_date IS NULL OR NEW.end_date IS NULL THEN
      RAISE EXCEPTION 'Trip start_date and end_date are required to publish' USING ERRCODE = '23514';
    END IF;
    IF NEW.capacity_rigs IS NULL THEN
      RAISE EXCEPTION 'Trip capacity_rigs is required to publish' USING ERRCODE = '23514';
    END IF;
    IF NEW.difficulty_text IS NULL OR btrim(NEW.difficulty_text) = '' THEN
      RAISE EXCEPTION 'Trip difficulty_text is required to publish' USING ERRCODE = '23514';
    END IF;
    IF NEW.meeting_location_label IS NULL OR btrim(NEW.meeting_location_label) = '' THEN
      RAISE EXCEPTION 'Trip meeting_location.label is required to publish' USING ERRCODE = '23514';
    END IF;
    IF NEW.comms_requirements_text IS NULL OR btrim(NEW.comms_requirements_text) = '' THEN
      RAISE EXCEPTION 'Trip comms_requirements_text is required to publish' USING ERRCODE = '23514';
    END IF;
    IF NEW.recommended_requirements_text IS NULL OR btrim(NEW.recommended_requirements_text) = '' THEN
      RAISE EXCEPTION 'Trip recommended_requirements_text is required to publish' USING ERRCODE = '23514';
    END IF;

    SELECT count(*) INTO organizer_count
    FROM trip_organizers
    WHERE trip_id = NEW.trip_id;

    IF organizer_count < 1 THEN
      RAISE EXCEPTION 'Trip must have at least one organizer to publish' USING ERRCODE = '23514';
    END IF;

    NEW.published_at := COALESCE(NEW.published_at, now());
    NEW.draft_visibility := NULL; -- no longer relevant after publish
  END IF;

  -- If status is changing to CANCELED, set canceled_at (idempotent allowed)
  IF (OLD.status <> 'CANCELED' AND NEW.status = 'CANCELED') THEN
    NEW.canceled_at := COALESCE(NEW.canceled_at, now());
    NEW.draft_visibility := NULL;
  END IF;

  RETURN NEW;
END;
$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_trips_enforce_transitions') THEN
    CREATE TRIGGER trg_trips_enforce_transitions
    BEFORE UPDATE ON trips
    FOR EACH ROW
    EXECUTE FUNCTION trips_enforce_transitions();
  END IF;
END $$;

-- =========================================================================
-- RSVP invariants:
-- - Allowed only when trip.status = PUBLISHED
-- - Capacity enforced strictly on YES (one rig per member)
-- =========================================================================
CREATE OR REPLACE FUNCTION enforce_rsvp_rules()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  t_status trip_status;
  t_capacity integer;
  current_yes integer;
  is_yes_transition boolean;
BEGIN
  SELECT status, capacity_rigs INTO t_status, t_capacity
  FROM trips
  WHERE trip_id = NEW.trip_id
  FOR UPDATE; -- serialize RSVP mutations per trip for capacity correctness

  IF t_status IS NULL THEN
    RAISE EXCEPTION 'Trip % does not exist', NEW.trip_id USING ERRCODE = '23503';
  END IF;

  IF t_status <> 'PUBLISHED' THEN
    RAISE EXCEPTION 'RSVPs are only allowed when trip is PUBLISHED (status=%)', t_status
      USING ERRCODE = '23514';
  END IF;

  -- Determine if this change consumes a rig slot.
  IF TG_OP = 'INSERT' THEN
    is_yes_transition := (NEW.response = 'YES');
  ELSE
    is_yes_transition := (OLD.response <> 'YES' AND NEW.response = 'YES');
  END IF;

  IF is_yes_transition THEN
    IF t_capacity IS NOT NULL THEN
      SELECT count(*) INTO current_yes
      FROM trip_rsvps
      WHERE trip_id = NEW.trip_id
        AND response = 'YES'
        AND NOT (TG_OP = 'UPDATE' AND member_id = NEW.member_id);

      IF current_yes >= t_capacity THEN
        RAISE EXCEPTION 'Trip capacity reached (% rigs)', t_capacity
          USING ERRCODE = '23514';
      END IF;
    END IF;
  END IF;

  NEW.updated_at := now();
  RETURN NEW;
END;
$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_trip_rsvps_enforce_rules') THEN
    CREATE TRIGGER trg_trip_rsvps_enforce_rules
    BEFORE INSERT OR UPDATE ON trip_rsvps
    FOR EACH ROW
    EXECUTE FUNCTION enforce_rsvp_rules();
  END IF;
END $$;
