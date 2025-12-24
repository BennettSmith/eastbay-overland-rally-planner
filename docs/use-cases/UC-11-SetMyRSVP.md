# UC-11 — SetMyRSVP

## Primary Actor

Member

## Goal

Set RSVP to YES, NO, or UNSET for a published trip.

## Preconditions

- Caller is authenticated.
- Target trip exists and is visible/accessible to the caller.

## Postconditions

- System state is updated as described.

---

## Main Success Flow

1. Actor requests setting their RSVP for a given `tripId` to one of `{YES, NO, UNSET}`.
2. System authenticates the caller.
3. System loads the trip by `tripId`.
4. System authorizes access:

   - Trip must be visible to the caller; if not, return `404 Not Found` (do not reveal existence).
5. System verifies the trip is `PUBLISHED`. If not, reject with `409 Conflict`.
6. System writes the member-owned RSVP record for `(tripId, callerMemberId)` using last-write-wins semantics:

   - `YES` consumes one rig slot and is rejected if capacity is reached.
   - `NO` and `UNSET` do not consume capacity.
7. System returns the caller’s current RSVP state (`myRsvp`).

---

## Alternate Flows

A1 — Change RSVP from YES to NO/UNSET

- **Condition:** Member previously had RSVP=YES.
- **Behavior:** System releases a rig slot and updates RSVP.
- **Outcome:** RSVP updated; capacity increases by 1.

A2 — Idempotent Update

- **Condition:** Member sets RSVP to the same value they already have.
- **Behavior:** System performs no state change.
- **Outcome:** Success response returned.

---

## Error Conditions

- `401 Unauthorized` — caller is not authenticated
- `404 Not Found` — trip does not exist OR is not visible to the caller
- `409 Conflict` — RSVP not allowed (e.g., trip is not published, capacity reached)
- `422 Unprocessable Entity` — invalid input values (format/range)
- `500 Internal Server Error` — unexpected failure

---

## Authorization Rules

- Caller must be an authenticated member.
- Caller may set only their own RSVP for the specified trip.
- Trip must be visible to the caller; if not, return `404 Not Found` (do not reveal existence).

## Domain Invariants Enforced

- RSVP is only allowed when Trip status is PUBLISHED.
- RSVP is owned by the member (member can only set their own RSVP).
- Capacity is enforced strictly when setting YES (YES consumes one rig slot).
- NO and UNSET do not consume capacity.
- No waitlist in v1; setting YES when capacity reached is rejected.
- Published trips must have an explicit capacity (>= 1 rig). Unlimited capacity (`null`) is not allowed for published trips.

---

## Output

- Success DTO containing the caller’s RSVP (`myRsvp`).

---

## API Notes

- Suggested endpoint: `PUT /trips/{tripId}/rsvp`
- Prefer returning a stable DTO shape; avoid leaking internal persistence fields.
- Mutating: **require** an `Idempotency-Key` header to safely handle retries.
- Use `PUT` semantics for last-write-wins on the member’s RSVP record.

---

## Notes

- Aligned with v1 guardrails: members-only, planning-focused, lightweight RSVP, artifacts referenced externally.
