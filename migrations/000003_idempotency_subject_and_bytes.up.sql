-- 000003_idempotency_subject_and_bytes.up.sql
--
-- Align idempotency persistence with the v1 application port:
-- - Keyed by authenticated subject (iss + sub) rather than member row FK
-- - Stores replayable responses as raw bytes (body + content type + status code)
--
-- Note: This intentionally replaces the v0/v1 draft `idempotency_keys` table shape.

DROP TABLE IF EXISTS idempotency_keys;

CREATE TABLE IF NOT EXISTS idempotency_keys (
  idempotency_key text NOT NULL,

  -- Authenticated actor (bearer JWT)
  subject_iss text NOT NULL,
  subject_sub text NOT NULL,

  -- Request identity
  method   text NOT NULL,
  route    text NOT NULL, -- normalized route template, e.g. "/trips/{tripId}/rsvp"
  body_hash text NOT NULL, -- sha256 hex; may be "" for meta records

  -- Stored record for replay
  status_code  integer NOT NULL,
  content_type text NOT NULL,
  body         bytea NOT NULL,
  created_at   timestamptz NOT NULL DEFAULT now(),
  expires_at   timestamptz NULL,

  PRIMARY KEY (idempotency_key, subject_iss, subject_sub, method, route, body_hash)
);

CREATE INDEX IF NOT EXISTS idx_idempotency_keys_expires_at ON idempotency_keys(expires_at);


