# UC-05 — SetTripDraftVisibility

## Primary Actor
Organizer (creator)

## Goal
Toggle a draft between private and public visibility.

## Preconditions
- Caller is authenticated.
- Target draft trip exists and is visible to the caller.

## Postconditions
- System state is updated as described.

---

## Main Success Flow
1. Actor submits a draft visibility change for a given `tripId` with `draftVisibility` in `{PRIVATE, PUBLIC}`.
2. System authenticates the caller.
3. System loads the trip by `tripId` and verifies `status = DRAFT`.
4. System authorizes the caller:
   - Caller must be the trip creator (`created_by_member_id`).
   - If not visible/authorized, return `404 Not Found` (do not reveal existence).
5. System updates `draftVisibility` to the requested value.
6. System returns the updated trip details.

---

## Alternate Flows
- None.

---

## Error Conditions
- `401 Unauthorized` — caller is not authenticated
- `404 Not Found` — trip does not exist OR is not visible to the caller
- `409 Conflict` — domain invariant violated
- `422 Unprocessable Entity` — invalid input values (format/range)
- `500 Internal Server Error` — unexpected failure

---

## Authorization Rules
- Caller must be an authenticated member.
- Only the trip creator (`created_by_member_id`) may change `draftVisibility`.
- For non-visible drafts, return `404 Not Found` (do not reveal existence).

## Domain Invariants Enforced
- Trip must be in DRAFT state.
- Only the creator may change draft visibility.
- draftVisibility must be one of PRIVATE or PUBLIC.

---

## Output
- Success DTO containing the updated trip.

---

## API Notes
- Suggested endpoint: `PUT /trips/{tripId}/draft-visibility`
- Prefer returning a stable DTO shape; avoid leaking internal persistence fields.
- Mutating: consider idempotency keys where duplicate submissions are plausible.

---

## Notes
- Aligned with v1 guardrails: members-only, planning-focused, lightweight RSVP, artifacts referenced externally.
