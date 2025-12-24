# UC-06 — PublishTrip

## Primary Actor
Organizer

## Goal
Publish a trip, enforcing required fields and returning prepared announcement copy for manual posting.

## Preconditions
- Caller is authenticated.
- Target trip exists and is visible/accessible to the caller.

## Postconditions
- System state is updated as described.

---

## Main Success Flow
1. Actor requests publish for a given `tripId`.
2. System authenticates the caller.
3. System loads the trip by `tripId`.
4. System authorizes and validates publish eligibility:
   - Trip must be visible to the caller (otherwise return `404 Not Found`).
   - Caller must be an organizer.
   - Trip must be in `DRAFT` state.
   - Trip must have `draftVisibility = PUBLIC` (private drafts are not publishable).
5. System validates required-at-publish fields are present and non-empty:
   - name
   - description
   - startDate
   - endDate
   - capacityRigs (must be >= 1)
   - difficultyText
   - meetingLocation
   - commsRequirementsText
   - recommendedRequirementsText
   - at least one organizer
6. System transitions the trip to `PUBLISHED`.
7. System returns the updated trip plus prepared announcement copy text for manual posting.

---

## Alternate Flows
A1 — Publish Already Published Trip
- **Condition:** Trip status is already `PUBLISHED`.
- **Behavior:** System performs no status change.
- **Outcome:** `200 OK` returned (idempotent). Announcement copy may still be returned (derived from current trip details).

A2 — Publish Private Draft
- **Condition:** Trip status is `DRAFT` and `draftVisibility = PRIVATE`.
- **Behavior:** System rejects publish (private drafts are not publishable).
- **Outcome:** `409 Conflict`.

A3 — Publish Canceled Trip
- **Condition:** Trip status is `CANCELED`.
- **Behavior:** System rejects publish.
- **Outcome:** `409 Conflict`.

---

## Error Conditions
- `401 Unauthorized` — caller is not authenticated
- `404 Not Found` — trip does not exist OR is not visible to the caller
- `409 Conflict` — publish is not allowed (e.g., private draft, canceled trip, missing required fields)
- `422 Unprocessable Entity` — invalid input values (format/range)
- `500 Internal Server Error` — unexpected failure

---

## Authorization Rules
- Caller must be an authenticated member.
- Trip must be visible to the caller; if not, return `404 Not Found` (do not reveal existence).
- Caller must be an organizer of the trip.
- Only `PUBLIC` drafts are publishable.

## Domain Invariants Enforced
- Trip must be in `DRAFT` state to publish.
- Trip must be a `PUBLIC` draft to publish (`draftVisibility = PUBLIC`).
- Required-at-publish fields must be present and non-empty: name, description, startDate, endDate, capacityRigs (>= 1), difficultyText, meetingLocation, commsRequirementsText, recommendedRequirementsText, at least one organizer.
- Once published, RSVP becomes allowed.
- At least one organizer must always exist.

---

## Output
- Success DTO containing:
  - the updated trip details
  - prepared announcement copy text for manual posting to the Google Group

---

## API Notes
- Suggested endpoint: `POST /trips/{tripId}/publish`
- Prefer returning a stable DTO shape; avoid leaking internal persistence fields.
- Publish should be idempotent (a publish request against an already published trip returns success and does not change state).

---

## Notes
- Aligned with v1 guardrails: members-only, planning-focused, lightweight RSVP, artifacts referenced externally.
