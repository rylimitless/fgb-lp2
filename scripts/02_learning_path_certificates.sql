-- Migration: Learning-path certificates
-- Run against the running DB:
--   docker compose exec db psql -U postgres -d example_db -f /path/to/02_learning_path_certificates.sql
-- Or (if the file isn't mounted into the container):
--   docker cp scripts/02_learning_path_certificates.sql $(docker compose ps -q db):/tmp/02.sql \
--     && docker compose exec db psql -U postgres -d example_db -f /tmp/02.sql
--
-- Idempotent: safe to re-run (everything uses IF NOT EXISTS).
-- This mirrors the block added to backend/database/schema.sql so that an
-- already-initialized database picks up the new table without a wipe.

-- ============================================================
-- 1. LEARNING PATH CERTIFICATES
-- ============================================================
-- Issued when a learner completes a learning path (all required courses done).
-- Mirrors `certificates` (course-level) but is keyed on learning_path_id.
CREATE TABLE IF NOT EXISTS learning_path_certificates (
  id bigserial primary key,
  user_id bigint not null references users(id) on delete cascade,
  learning_path_id bigint not null references learning_paths(id) on delete cascade,
  issued_at timestamptz not null default now(),
  certificate_code text not null unique,          -- UUID-based unique code for verification
  score_pct numeric(5,2) not null default 0,
  tier text not null default 'bronze' check (tier in ('bronze', 'silver', 'gold')),
  unique(user_id, learning_path_id)
);

CREATE INDEX IF NOT EXISTS idx_lp_certificates_user  ON learning_path_certificates(user_id);
CREATE INDEX IF NOT EXISTS idx_lp_certificates_path  ON learning_path_certificates(learning_path_id);
CREATE INDEX IF NOT EXISTS idx_lp_certificates_code  ON learning_path_certificates(certificate_code);

-- ============================================================
-- Verify
-- ============================================================
SELECT 'learning_path_certificates' as object, count(*) as rows FROM learning_path_certificates;
