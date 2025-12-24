-- File: 06_seed_dev_optional.sql
-- Development / demo seed data for East Bay Overland
-- Safe to run multiple times (uses deterministic UUIDs via INSERT ... SELECT WHERE NOT EXISTS patterns)
-- Assumes schema has already been created.

-- =========================================================================
-- Members
-- =========================================================================
INSERT INTO members (member_id, subject_iss, subject_sub, display_name, email)
SELECT *
FROM (VALUES
  ('11111111-1111-1111-1111-111111111111'::uuid, 'https://auth.dev.example', 'sub-alice', 'Alice Trailboss', 'alice@example.com'),
  ('22222222-2222-2222-2222-222222222222'::uuid, 'https://auth.dev.example', 'sub-bob',   'Bob Rockcrawler', 'bob@example.com'),
  ('33333333-3333-3333-3333-333333333333'::uuid, 'https://auth.dev.example', 'sub-cara',  'Cara Navigator',  'cara@example.com'),
  ('44444444-4444-4444-4444-444444444444'::uuid, 'https://auth.dev.example', 'sub-dan',   'Dan Overlander',  'dan@example.com')
) AS v(member_id, subject_iss, subject_sub, display_name, email)
WHERE NOT EXISTS (
  SELECT 1 FROM members m WHERE m.member_id = v.member_id
);

-- Vehicle profiles
INSERT INTO member_vehicle_profiles (
  member_id, make, model, tire_size, lift_lockers, fuel_range, recovery_gear, ham_radio_call_sign, notes
)
SELECT *
FROM (VALUES
  ('11111111-1111-1111-1111-111111111111'::uuid, 'Toyota', '4Runner', '285/70R17', '2in lift, rear locker', '300mi', 'Winch, traction boards', 'K6ALC', 'Trip lead'),
  ('22222222-2222-2222-2222-222222222222'::uuid, 'Jeep', 'Wrangler Rubicon', '35in', 'Front+rear lockers', '250mi', 'Winch, hi-lift', NULL, 'Hardcore trails'),
  ('33333333-3333-3333-3333-333333333333'::uuid, 'Toyota', 'Tacoma', '33in', 'Rear locker', '320mi', 'Traction boards', NULL, 'New to group'),
  ('44444444-4444-4444-4444-444444444444'::uuid, 'Ford', 'Bronco', '35in', 'Sasquatch pkg', '280mi', 'Winch', 'W6DAN', 'Good comms')
) AS v(
  member_id, make, model, tire_size, lift_lockers, fuel_range, recovery_gear, ham_radio_call_sign, notes
)
WHERE NOT EXISTS (
  SELECT 1 FROM member_vehicle_profiles p WHERE p.member_id = v.member_id
);

-- =========================================================================
-- Trips
-- =========================================================================
-- Draft (private)
INSERT INTO trips (
  trip_id, name, description, status, draft_visibility
)
SELECT *
FROM (VALUES
  (
    'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'::uuid,
    'Mendocino NF Shakedown',
    'Early-season shakedown trip to test rigs and comms.',
    'DRAFT'::trip_status,
    'PRIVATE'::draft_visibility
  )
) AS v(trip_id, name, description, status, draft_visibility)
WHERE NOT EXISTS (
  SELECT 1 FROM trips t WHERE t.trip_id = v.trip_id
);

-- Published trip
INSERT INTO trips (
  trip_id,
  name,
  description,
  start_date,
  end_date,
  status,
  capacity_rigs,
  difficulty_text,
  meeting_location_label,
  meeting_location_address,
  comms_requirements_text,
  recommended_requirements_text,
  published_at
)
SELECT *
FROM (VALUES
  (
    'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'::uuid,
    'Death Valley Explorer Loop',
    'Multi-day overland loop through Death Valley NP backcountry.',
    DATE '2026-02-12',
    DATE '2026-02-16',
    'PUBLISHED'::trip_status,
    5,
    'Moderate: sand, rocks, long days',
    'Furnace Creek Gas Station',
    'Furnace Creek Rd, Death Valley, CA',
    'GMRS or HAM required; channel assigned night before',
    'Full-size spare, 300+ mile range, recovery gear',
    now()
  )
) AS v(
  trip_id, name, description, start_date, end_date, status, capacity_rigs,
  difficulty_text, meeting_location_label, meeting_location_address,
  comms_requirements_text, recommended_requirements_text, published_at
)
WHERE NOT EXISTS (
  SELECT 1 FROM trips t WHERE t.trip_id = v.trip_id
);

-- =========================================================================
-- Organizers
-- =========================================================================
INSERT INTO trip_organizers (trip_id, member_id)
SELECT *
FROM (VALUES
  ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'::uuid, '11111111-1111-1111-1111-111111111111'::uuid),
  ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'::uuid, '11111111-1111-1111-1111-111111111111'::uuid),
  ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'::uuid, '22222222-2222-2222-2222-222222222222'::uuid)
) AS v(trip_id, member_id)
WHERE NOT EXISTS (
  SELECT 1 FROM trip_organizers o
  WHERE o.trip_id = v.trip_id AND o.member_id = v.member_id
);

-- =========================================================================
-- Trip Artifacts
-- =========================================================================
INSERT INTO trip_artifacts (trip_id, type, title, url, sort_order)
SELECT *
FROM (VALUES
  (
    'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'::uuid,
    'GPX'::artifact_type,
    'Death Valley Explorer Loop',
    'https://example.com/artifacts/death-valley-loop.gpx',
    1
  ),
  (
    'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'::uuid,
    'DOCUMENT'::artifact_type,
    'Trip Itinerary PDF',
    'https://example.com/artifacts/death-valley-itinerary.pdf',
    2
  )
) AS v(trip_id, type, title, url, sort_order)
WHERE NOT EXISTS (
  SELECT 1 FROM trip_artifacts a
  WHERE a.trip_id = v.trip_id AND a.title = v.title
);

-- =========================================================================
-- RSVPs
-- =========================================================================
INSERT INTO trip_rsvps (trip_id, member_id, response)
SELECT *
FROM (VALUES
  ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'::uuid, '11111111-1111-1111-1111-111111111111'::uuid, 'YES'::rsvp_response),
  ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'::uuid, '22222222-2222-2222-2222-222222222222'::uuid, 'YES'::rsvp_response),
  ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'::uuid, '33333333-3333-3333-3333-333333333333'::uuid, 'UNSET'::rsvp_response)
) AS v(trip_id, member_id, response)
WHERE NOT EXISTS (
  SELECT 1 FROM trip_rsvps r
  WHERE r.trip_id = v.trip_id AND r.member_id = v.member_id
);

-- End of seed data
