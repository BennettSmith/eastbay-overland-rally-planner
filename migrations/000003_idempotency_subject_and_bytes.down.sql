-- 000003_idempotency_subject_and_bytes.down.sql
--
-- Best-effort rollback: drop the subject-keyed idempotency table.
-- (We do not recreate the previous shape automatically.)

DROP TABLE IF EXISTS idempotency_keys;


