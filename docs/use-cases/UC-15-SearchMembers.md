# UC-15 — SearchMembers

## Primary Actor

Member

## Goal

Search members by name (used for adding organizers).

## Preconditions

- Caller is authenticated.

## Postconditions

- Trip/member data is returned. No state is modified.

---

## Main Success Flow

1. Actor requests member search with query `q`.
2. System authenticates the caller.
3. System validates inputs:

   - `q` must be at least 3 characters.
4. System searches **active** members by display name using a “full-text-ish” match:

   - Case-insensitive.
   - Tokenized: supports multi-word queries (e.g., `john smith`).
   - Matches can be order-insensitive; results may include partial matches depending on implementation.
   - Email fields are not searched.
5. System orders results by `displayName` ascending.
6. System returns a list of privacy-safe directory entries (no email fields).

---

## Alternate Flows

- None.

---

## Error Conditions

- `401 Unauthorized` — caller is not authenticated
- `422 Unprocessable Entity` — invalid search query (e.g., `q` too short)
- `500 Internal Server Error` — unexpected failure

---

## Authorization Rules

- Caller must be an authenticated member.
- Any authenticated member may search the member directory.

---

## Output

- Success DTO containing a list of minimal member directory entries.
- Results must not expose email addresses.

---

## API Notes

- Suggested endpoint: `GET /members/search?q={query}`
- Prefer returning a stable DTO shape; avoid leaking internal persistence fields.
- Read-only: safe and cacheable (where appropriate).

---

## Notes

- Aligned with v1 guardrails: members-only, planning-focused, lightweight RSVP, artifacts referenced externally.
