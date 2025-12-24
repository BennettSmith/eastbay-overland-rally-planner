# UC-03 — CreateTripDraft

## Primary Actor
Member (creator)

## Goal
Create a new trip in `DRAFT` state. Incomplete data is allowed.

## Preconditions
- Caller is authenticated.

## Postconditions
- A new draft trip exists.
- The caller is recorded as the trip creator (`created_by_member_id`) and is the initial organizer.

---

## Main Success Flow
1. Actor submits a create-draft request (optional partial trip fields).
2. System authenticates the caller.
3. System creates a new `Trip` with:
   - `status = DRAFT`
   - `draftVisibility = PRIVATE` (unless explicitly set to `PUBLIC`)
   - `created_by_member_id = caller`
4. System ensures the creator is included as an organizer (initial organizer).
5. System returns the created trip identifiers/details.

---

## Alternate Flows
- None.

---

## Error Conditions
- `401 Unauthorized` — caller is not authenticated
- `409 Conflict` — domain invariant violated
- `422 Unprocessable Entity` — invalid input values (format/range)
- `500 Internal Server Error` — unexpected failure

---

## Authorization Rules
- Caller must be an authenticated member.
- Any authenticated member may create a new trip draft; the caller becomes the creator and initial organizer.

## Domain Invariants Enforced
- Trip status is initialized to `DRAFT`.
- `draftVisibility` defaults to `PRIVATE` unless explicitly set otherwise.
- `created_by_member_id` is set to the caller and is immutable.
- At least one organizer must exist; the creator becomes the initial organizer.

---

## Output
- Success DTO containing the created trip (at minimum the new `tripId`).

---

## API Notes
- Suggested endpoint: `POST /trips`
- Prefer returning a stable DTO shape; avoid leaking internal persistence fields.
- Mutating: consider idempotency keys where duplicate submissions are plausible.

---

## Notes
- Aligned with v1 guardrails: members-only, planning-focused, lightweight RSVP, artifacts referenced externally.
