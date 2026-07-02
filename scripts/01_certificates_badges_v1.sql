-- Migration: Add certificates and badges (certificates_badges_v1)
-- Run against the running DB: docker compose exec db psql -U postgres -d fgb_lp -f /path/to/this/file
-- Or: docker cp this/file fgb-db1:/tmp/ && docker compose exec db psql -U postgres -d fgb_lp -f /tmp/01_certificates_badges_v1.sql

-- ============================================================
-- 1. CERTIFICATES
-- ============================================================
CREATE TABLE IF NOT EXISTS certificates (
  id bigserial primary key,
  user_id bigint not null references users(id) on delete cascade,
  course_id bigint not null references courses(id) on delete cascade,
  issued_at timestamptz not null default now(),
  certificate_code text not null unique,
  score_pct numeric(5,2) not null default 0,
  tier text not null default 'bronze' check (tier in ('bronze', 'silver', 'gold')),
  unique(user_id, course_id)
);

CREATE INDEX IF NOT EXISTS idx_certificates_user ON certificates(user_id);
CREATE INDEX IF NOT EXISTS idx_certificates_course ON certificates(course_id);
CREATE INDEX IF NOT EXISTS idx_certificates_code ON certificates(certificate_code);

-- ============================================================
-- 2. BADGE DEFINITIONS
-- ============================================================
-- Drop the old table if it lacks the code column (migration from earlier agent run)
-- We'll recreate it properly
DO $$
BEGIN
  -- Check if badge_definitions exists and if it lacks the 'code' column
  IF EXISTS (
    SELECT 1 FROM information_schema.tables WHERE table_name = 'badge_definitions'
  ) AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns WHERE table_name = 'badge_definitions' AND column_name = 'code'
  ) THEN
    -- Old table without 'code' column — drop it (cascade will drop badge_awards too)
    DROP TABLE IF EXISTS badge_awards CASCADE;
    DROP TABLE IF EXISTS badge_definitions CASCADE;
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS badge_definitions (
  id bigserial primary key,
  code text not null unique,
  name text not null,
  description text not null default '',
  icon text not null default 'star',
  tier text not null default 'bronze' check (tier in ('bronze', 'silver', 'gold')),
  category text not null default 'achievement' check (category in ('achievement', 'streak', 'milestone', 'speed', 'skill')),
  criteria jsonb not null default '{}',
  created_at timestamptz not null default now()
);

-- ============================================================
-- 3. BADGE AWARDS
-- ============================================================
CREATE TABLE IF NOT EXISTS badge_awards (
  id bigserial primary key,
  user_id bigint not null references users(id) on delete cascade,
  badge_id bigint not null references badge_definitions(id) on delete cascade,
  earned_at timestamptz not null default now(),
  metadata jsonb not null default '{}',
  unique(user_id, badge_id)
);

CREATE INDEX IF NOT EXISTS idx_badge_awards_user ON badge_awards(user_id);
CREATE INDEX IF NOT EXISTS idx_badge_awards_badge ON badge_awards(badge_id);

-- ============================================================
-- 4. SEED BADGE DEFINITIONS
-- ============================================================
INSERT INTO badge_definitions (code, name, description, icon, tier, category, criteria) VALUES
  ('first_course', 'First Steps', 'Complete your first course', 'footprints', 'bronze', 'milestone', '{"type":"courses_completed","count":1}'),
  ('five_courses', 'Scholar', 'Complete 5 courses', 'graduation-cap', 'silver', 'milestone', '{"type":"courses_completed","count":5}'),
  ('ten_courses', 'Master Learner', 'Complete 10 courses', 'award', 'gold', 'milestone', '{"type":"courses_completed","count":10}'),
  ('streak_3', 'Consistent', 'Maintain a 3-day streak', 'flame', 'bronze', 'streak', '{"type":"streak","count":3}'),
  ('streak_7', 'Dedicated', 'Maintain a 7-day streak', 'flame', 'silver', 'streak', '{"type":"streak","count":7}'),
  ('streak_30', 'Unstoppable', 'Maintain a 30-day streak', 'flame', 'gold', 'streak', '{"type":"streak","count":30}'),
  ('perfect_score', 'Perfect Score', 'Score 100% on any course', 'target', 'gold', 'achievement', '{"type":"perfect_score","count":1}'),
  ('gold_course', 'Gold Standard', 'Score 90%+ on a course', 'star', 'gold', 'achievement', '{"type":"score_above","threshold":90}'),
  ('speed_demon', 'Speed Demon', 'Complete a course in 1 day', 'zap', 'silver', 'speed', '{"type":"speed_days","max_days":1}'),
  ('quick_learner', 'Quick Learner', 'Complete a course in 3 days', 'clock', 'bronze', 'speed', '{"type":"speed_days","max_days":3}'),
  ('total_xp_1000', 'XP Apprentice', 'Earn 1,000 total XP', 'trending-up', 'bronze', 'achievement', '{"type":"total_xp","count":1000}'),
  ('total_xp_5000', 'XP Expert', 'Earn 5,000 total XP', 'trending-up', 'silver', 'achievement', '{"type":"total_xp","count":5000}')
ON CONFLICT (code) DO NOTHING;

-- ============================================================
-- Verify
-- ============================================================
SELECT 'certificates table' as object, count(*) as rows FROM certificates
UNION ALL
SELECT 'badge_definitions table', count(*) FROM badge_definitions
UNION ALL
SELECT 'badge_awards table', count(*) FROM badge_awards;
