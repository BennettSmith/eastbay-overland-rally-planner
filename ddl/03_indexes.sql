-- File: 03_indexes.sql

-- Members lookup by subject and search by name/email
CREATE INDEX IF NOT EXISTS idx_members_subject ON members(subject_iss, subject_sub);
CREATE INDEX IF NOT EXISTS idx_members_display_name ON members USING gin (to_tsvector('simple', display_name));
CREATE INDEX IF NOT EXISTS idx_members_email ON members(email);

-- Trips listing (published + public drafts) and sorting by start date
CREATE INDEX IF NOT EXISTS idx_trips_status_start_date ON trips(status, start_date);
CREATE INDEX IF NOT EXISTS idx_trips_draft_visibility ON trips(draft_visibility) WHERE status = 'DRAFT';

-- Organizers
CREATE INDEX IF NOT EXISTS idx_trip_organizers_member ON trip_organizers(member_id);

-- Artifacts
CREATE INDEX IF NOT EXISTS idx_trip_artifacts_trip ON trip_artifacts(trip_id);

-- RSVPs
CREATE INDEX IF NOT EXISTS idx_trip_rsvps_trip_response ON trip_rsvps(trip_id, response);
CREATE INDEX IF NOT EXISTS idx_trip_rsvps_member ON trip_rsvps(member_id);

-- Idempotency
CREATE INDEX IF NOT EXISTS idx_idempotency_keys_expires_at ON idempotency_keys(expires_at);
