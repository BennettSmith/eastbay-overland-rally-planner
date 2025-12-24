-- File: 05_views.sql

-- Convenience view for trip listing with attendingRigs count.
CREATE OR REPLACE VIEW v_trip_summary AS
SELECT
  t.trip_id,
  t.name,
  t.start_date,
  t.end_date,
  t.status,
  t.draft_visibility,
  t.capacity_rigs,
  COALESCE(r.attending_rigs, 0) AS attending_rigs,
  t.created_at,
  t.updated_at
FROM trips t
LEFT JOIN (
  SELECT trip_id, count(*) AS attending_rigs
  FROM trip_rsvps
  WHERE response = 'YES'
  GROUP BY trip_id
) r ON r.trip_id = t.trip_id;

-- Convenience view for RSVP summary lists.
-- (Application can still filter by trip visibility rules in queries.)
CREATE OR REPLACE VIEW v_trip_rsvp_summary AS
SELECT
  t.trip_id,
  t.capacity_rigs,
  COALESCE(a.attending_rigs, 0) AS attending_rigs
FROM trips t
LEFT JOIN (
  SELECT trip_id, count(*) AS attending_rigs
  FROM trip_rsvps
  WHERE response = 'YES'
  GROUP BY trip_id
) a ON a.trip_id = t.trip_id;
