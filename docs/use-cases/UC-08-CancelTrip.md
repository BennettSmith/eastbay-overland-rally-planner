# UC-08 — CancelTrip

## Primary Actor
Organizer

## Goal
Cancel a trip. RSVPs are disabled and the trip becomes read-only.

## Preconditions
- Caller is authenticated.
- Target trip exists and is visible/accessible to the caller.

## Postconditions
- System state is updated as described.

---

## Main Success Flow
1. Actor requests cancellation for a given `tripId`.
2. System authenticates the caller.
3. System loads the trip by `tripId`.
4. System authorizes access:
   - Trip must be visible to the caller; if not, return `404 Not Found` (do not reveal existence).
   - Caller must be an organizer; if not, return `404 Not Found`.
5. System transitions the trip status to `CANCELED` (allowed from `DRAFT` or `PUBLISHED`).
6. System returns the updated trip details.

---

## Alternate Flows
A1 — Cancel Already Canceled Trip
- **Condition:** Trip status is already `CANCELED`.
- **Behavior:** System performs no additional changes.
- **Outcome:** `200 OK` returned (idempotent).

---

## Error Conditions
- `401 Unauthorized` — caller is not authenticated
- `404 Not Found` — trip does not exist OR is not visible to the caller
- `409 Conflict` — cancel is not allowed (e.g., domain invariant violated)
- `422 Unprocessable Entity` — invalid input values (format/range)
- `500 Internal Server Error` — unexpected failure

---

## Authorization Rules
- Caller must be an authenticated member.
- Trip must be visible to the caller; if not, return `404 Not Found` (do not reveal existence).
- Caller must be an organizer of the trip; if not, return `404 Not Found`.

## Domain Invariants Enforced
- Only organizers may cancel a trip.
- Trip status transitions to CANCELED (from DRAFT or PUBLISHED).
- After cancelation, RSVP mutations are disabled.
- Cancelation is idempotent (canceling an already-canceled trip has no additional effect).
- No changes are allowed after cancelation; trips cannot be un-canceled.

---

## Output
- Success DTO containing the updated trip.

---

## API Notes
- Suggested endpoint: `POST /trips/{tripId}/cancel`
- Prefer returning a stable DTO shape; avoid leaking internal persistence fields.
- Cancellation is inherently idempotent; an idempotency key is optional.

---

## Notes
- Aligned with v1 guardrails: members-only, planning-focused, lightweight RSVP, artifacts referenced externally.
