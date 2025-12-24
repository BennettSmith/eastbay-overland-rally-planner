-- File: 02_tables.sql

-- =========================================================================
-- Members
-- =========================================================================
CREATE TABLE IF NOT EXISTS members (
  member_id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Binding to authenticated subject from bearer JWT (see UC-17/UC-18).
  -- Use (subject_iss, subject_sub) uniqueness to support multiple issuers if needed.
  subject_iss           text NOT NULL,
  subject_sub           text NOT NULL,

  display_name          text NOT NULL,
  email                 citext NOT NULL,
  group_alias_email     citext NULL,

  -- Soft admin controls (optional / future-proof):
  is_active             boolean NOT NULL DEFAULT true,

  created_at            timestamptz NOT NULL DEFAULT now(),
  updated_at            timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT members_subject_unique UNIQUE (subject_iss, subject_sub),
  CONSTRAINT members_email_unique UNIQUE (email)
);

CREATE TABLE IF NOT EXISTS member_vehicle_profiles (
  member_id           uuid PRIMARY KEY REFERENCES members(member_id) ON DELETE CASCADE,

  make                text NULL,
  model               text NULL,
  tire_size           text NULL,
  lift_lockers        text NULL,
  fuel_range          text NULL,
  recovery_gear       text NULL,
  ham_radio_call_sign text NULL,
  notes               text NULL,

  updated_at          timestamptz NOT NULL DEFAULT now()
);

-- =========================================================================
-- Trips
-- =========================================================================
CREATE TABLE IF NOT EXISTS trips (
  trip_id                     uuid PRIMARY KEY DEFAULT gen_random_uuid(),

  name                        text NULL,
  description                 text NULL,
  start_date                  date NULL,
  end_date                    date NULL,

  status                      trip_status NOT NULL DEFAULT 'DRAFT',
  draft_visibility            draft_visibility NULL,

  capacity_rigs               integer NULL CHECK (capacity_rigs IS NULL OR capacity_rigs >= 0),
  difficulty_text             text NULL,

  -- Meeting location (embedded as nullable columns)
  meeting_location_label      text NULL,
  meeting_location_address    text NULL,
  meeting_location_latitude   double precision NULL,
  meeting_location_longitude  double precision NULL,

  comms_requirements_text     text NULL,
  recommended_requirements_text text NULL,

  -- Publish/cancel bookkeeping (useful for idempotent behavior and UI)
  published_at                timestamptz NULL,
  canceled_at                 timestamptz NULL,

  created_at                  timestamptz NOT NULL DEFAULT now(),
  updated_at                  timestamptz NOT NULL DEFAULT now(),

  -- Draft visibility is only relevant for DRAFTs (v1 domain model).
  CONSTRAINT trips_draft_visibility_consistency CHECK (
    (status = 'DRAFT' AND draft_visibility IS NOT NULL)
    OR (status <> 'DRAFT' AND draft_visibility IS NULL)
  ),

  CONSTRAINT trips_dates_consistency CHECK (
    (start_date IS NULL OR end_date IS NULL) OR (start_date <= end_date)
  ),

  CONSTRAINT trips_meeting_location_latlon_consistency CHECK (
    (meeting_location_latitude IS NULL AND meeting_location_longitude IS NULL)
    OR (meeting_location_latitude IS NOT NULL AND meeting_location_longitude IS NOT NULL)
  )
);

-- Many-to-many: trips <-> organizers (members)
CREATE TABLE IF NOT EXISTS trip_organizers (
  trip_id     uuid NOT NULL REFERENCES trips(trip_id) ON DELETE CASCADE,
  member_id   uuid NOT NULL REFERENCES members(member_id) ON DELETE RESTRICT,
  added_at    timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (trip_id, member_id)
);

-- Trip artifacts (externally hosted; referenced by URL)
CREATE TABLE IF NOT EXISTS trip_artifacts (
  trip_artifact_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  trip_id          uuid NOT NULL REFERENCES trips(trip_id) ON DELETE CASCADE,
  type             artifact_type NOT NULL,
  title            text NOT NULL,
  url              text NOT NULL, -- validate as URI at application layer

  sort_order       integer NOT NULL DEFAULT 0,

  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now()
);

-- =========================================================================
-- RSVPs (member-owned, one vehicle per member per trip)
-- =========================================================================
CREATE TABLE IF NOT EXISTS trip_rsvps (
  trip_id        uuid NOT NULL REFERENCES trips(trip_id) ON DELETE CASCADE,
  member_id      uuid NOT NULL REFERENCES members(member_id) ON DELETE CASCADE,

  response       rsvp_response NOT NULL DEFAULT 'UNSET',
  updated_at     timestamptz NOT NULL DEFAULT now(),

  PRIMARY KEY (trip_id, member_id)
);

-- =========================================================================
-- Optional: Idempotency keys for mutating endpoints (recommended in use cases)
-- =========================================================================
CREATE TABLE IF NOT EXISTS idempotency_keys (
  idempotency_key   text NOT NULL,
  actor_member_id   uuid NOT NULL REFERENCES members(member_id) ON DELETE CASCADE,
  scope             text NOT NULL, -- e.g., "POST /trips", "PUT /trips/{id}/rsvp"
  request_hash      text NULL,     -- optional: to detect mismatched retries
  response_body     jsonb NULL,    -- optional: cached response for safe retries
  created_at        timestamptz NOT NULL DEFAULT now(),
  expires_at        timestamptz NULL,

  PRIMARY KEY (idempotency_key, actor_member_id, scope)
);
