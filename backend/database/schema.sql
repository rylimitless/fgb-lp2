CREATE EXTENSION IF NOT EXISTS vector;

create table if not exists users (
  id bigserial primary key,

  email text not null unique,
  password_hash text not null,

  name text not null,

  role text not null default 'end user'
    check (role in ('end user','content creator', 'admin', 'approver', 'manager', 'auditor')),

  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists sessions (
  id bigserial primary key,
  user_id bigint not null references users(id) on delete cascade,
  token text not null unique,
  expires_at timestamptz not null,
  created_at timestamptz not null default now()
);

-- Many-to-many user-role assignments
create table if not exists user_roles (
  user_id bigint not null references users(id) on delete cascade,
  role text not null,
  primary key (user_id, role)
);

-- A flat list of every action someone could take
CREATE TABLE IF NOT EXISTS permissions (
  id   BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE   -- e.g. "users:read", "content:create", "reports:export"
);

-- Which permissions belong to which role
CREATE TABLE IF NOT EXISTS role_permissions (
  role        TEXT NOT NULL,
  permission_id BIGINT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
  PRIMARY KEY (role, permission_id)
);


INSERT INTO permissions (name) VALUES
  ('users:read'),
  ('users:write'),
  ('users:delete'),
  ('content:create'),
  ('content:edit'),
  ('content:delete'),
  ('content:approve'),
  ('reports:view'),
  ('settings:manage');

-- Admin gets everything
INSERT INTO role_permissions (role, permission_id)
SELECT 'admin', id FROM permissions;



create table documents (
  id bigserial primary key,
  title text not null,
  file_path text not null,
  status text not null default 'uploaded'
    check (status in ('uploaded', 'processing', 'ready', 'failed')),
  uploaded_by bigint references users(id),
  total_chunks int not null default 0,
  chunks_done int not null default 0,
  created_at timestamptz not null default now()
);

alter table documents add column approved boolean default false;
alter table documents add column error_message text default '';
alter table documents add column review_status text default 'pending';
alter table documents add column review_notes text default '';


create table document_chunks (
  id bigserial primary key,
  document_id bigint not null references documents(id) on delete cascade,

  chunk_index int not null,
  content text not null,

  page_number int,
  source_label text,

  embedding vector(384), -- 384-dim (all-MiniLM-L6-v2 via local fastembed service)

  created_at timestamptz not null default now(),

  unique(document_id, chunk_index)
);

-- Content Studio: courses generated from documents
create table courses (
  id bigserial primary key,
  title text not null,
  description text not null default '',
  created_by bigint not null references users(id) on delete cascade,
  source_doc_ids bigint[] default '{}',
  status text not null default 'draft'
    check (status in ('draft', 'published', 'archived')),
  settings jsonb not null default '{}',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);


alter table courses add column department varchar(255);
alter table courses add column approved boolean default false;
alter table courses add column review_status text default 'pending';
alter table courses add column review_notes text default '';
alter table courses add column approved_by bigint references users(id);

-- Departments: organisational grouping for users and courses
create table if not exists departments (
  id bigserial primary key,
  name text not null unique,
  created_at timestamptz not null default now()
);

-- Many-to-many: users assigned to departments
create table if not exists user_departments (
  user_id bigint not null references users(id) on delete cascade,
  department_id bigint not null references departments(id) on delete cascade,
  primary key (user_id, department_id)
);
create index if not exists idx_user_departments_dept
  on user_departments(department_id);

-- Migrations: add module support
create table if not exists modules (
  id bigserial primary key,
  course_id bigint not null references courses(id) on delete cascade,
  title text not null,
  description text not null default '',
  sort_order int not null default 0,
  -- Per-module generation status for the staged course builder.
  -- pending = outline-only, no content/items yet
  -- generating = a module-generation job is currently running
  -- ready = content + items have been generated
  -- failed = the last generation attempt failed
  status text not null default 'pending'
    check (status in ('pending', 'generating', 'ready', 'failed')),
  -- Per-module question type allowlist (null/empty = allow all types).
  -- Set by the user during outline review before generation.
  question_types jsonb,
  -- AI-reported limitations captured during generation: which requested
  -- question types were skipped because the section material didn't fit
  -- them, and why. Surfaced to the user as warnings after generation.
  limitations jsonb,
  created_at timestamptz not null default now()
);

create table if not exists course_items (
  id bigserial primary key,
  course_id bigint not null references courses(id) on delete cascade,
  module_id bigint references modules(id) on delete set null,
  item_type text not null
    check (item_type in ('mc', 'sa', 'tf', 'fb', 'dd', 'essay')),
  sort_order int not null default 0,
  data jsonb not null default '{}',
  created_at timestamptz not null default now()
);

alter table course_items add column if not exists module_id bigint references modules(id) on delete set null;

-- Drop old item_type check constraint and add new one with all interaction types
do $$
declare
  constraint_name text;
begin
  select con.conname into constraint_name
  from pg_constraint con
  join pg_class rel on rel.oid = con.conrelid
  where rel.relname = 'course_items'
    and con.contype = 'c'
    and pg_get_constraintdef(con.oid) like '%item_type%';
  if constraint_name is not null then
    execute 'alter table course_items drop constraint ' || constraint_name;
  end if;
end $$;

alter table course_items add constraint course_items_item_type_check
  check (item_type in ('content','mc','ma','tf','fb','sa','matching','drag_sort','hotspot','sequence','scale'));

-- Adaptive Practice Room: stores per-user learning preferences
create table learning_preferences (
  user_id bigint primary key references users(id) on delete cascade,
  learning_style text not null default 'mixed'
    check (learning_style in ('visual', 'textual', 'interactive', 'mixed')),
  difficulty_level text not null default 'intermediate'
    check (difficulty_level in ('beginner', 'intermediate', 'advanced', 'adaptive')),
  preferred_topics text[] default '{}',
  theta double precision not null default 0.0,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

ALTER TABLE learning_preferences ADD COLUMN IF NOT EXISTS theta double precision NOT NULL DEFAULT 0.0;


-- Adaptive Practice Room: tracks practice sessions
create table if not exists practice_sessions (
  id bigserial primary key,
  user_id bigint not null references users(id) on delete cascade,
  topic text not null default '',
  question_count int not null default 0,
  correct_count int not null default 0,
  score_pct numeric(5,2) not null default 0,
  started_at timestamptz not null default now(),
  completed_at timestamptz
);

-- Guided Lesson Player: tracks per-user progress through a course
create table if not exists lesson_progress (
  id bigserial primary key,
  user_id bigint not null references users(id) on delete cascade,
  course_id bigint not null references courses(id) on delete cascade,
  current_module int not null default 0,
  completed boolean not null default false,
  score_pct numeric(5,2) not null default 0,
  started_at timestamptz not null default now(),
  completed_at timestamptz,
  unique(user_id, course_id)
);

-- Guided Lesson Player: per-user per-item answers and correctness
create table if not exists item_progress (
  id bigserial primary key,
  user_id bigint not null references users(id) on delete cascade,
  course_id bigint not null references courses(id) on delete cascade,
  item_id bigint not null references course_items(id) on delete cascade,
  answer jsonb not null default 'null',
  is_correct boolean,
  answered_at timestamptz not null default now(),
  unique(user_id, item_id)
);
create index if not exists idx_item_progress_user_course
  on item_progress(user_id, course_id);

-- Notifications: system notifications for users
create table if not exists notifications (
  id bigserial primary key,
  user_id bigint references users(id) on delete cascade,
  title text not null,
  message text not null default '',
  link text not null default '',
  is_read boolean not null default false,
  created_at timestamptz not null default now()
);
create index if not exists idx_notifications_user_unread
  on notifications(user_id, is_read) where is_read = false;

-- Course Generation Jobs: persists generation state across restarts
create table if not exists course_generation_jobs (
  id text primary key,
  status text not null default 'pending'
    check (status in ('pending', 'running', 'completed', 'failed')),
  -- Which stage of the staged builder this job represents:
  -- full = legacy one-shot (outline + all modules)
  -- outline = outline only, stops after module rows are created
  -- module = single module content/items generation
  stage text not null default 'full'
    check (stage in ('full', 'outline', 'module')),
  request jsonb not null,
  steps jsonb not null default '[]',
  modules jsonb not null default '[]',
  result jsonb,
  error text,
  course_id bigint references courses(id) on delete set null,
  -- For stage='module' jobs, which module row this job is filling in.
  module_id bigint references modules(id) on delete set null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

-- Coach queries: tracks GIA AI Coach usage for analytics
create table if not exists coach_queries (
  id bigserial primary key,
  user_id bigint references users(id) on delete set null,
  question text not null,
  sources_count int not null default 0,
  created_at timestamptz not null default now()
);
create index if not exists idx_coach_queries_created_at on coach_queries(created_at);

-- Audit log: single table capturing all significant actions for admin review
create table if not exists audit_log (
  id bigserial primary key,
  user_id bigint references users(id) on delete set null,
  action text not null,
  details jsonb not null default '{}',
  created_at timestamptz not null default now()
);
create index if not exists idx_audit_log_created_at on audit_log(created_at desc);
create index if not exists idx_audit_log_user_id on audit_log(user_id);
create index if not exists idx_audit_log_action on audit_log(action);

-- Streaks: daily engagement tracking for gamification
create table if not exists user_streaks (
  user_id bigint not null references users(id) on delete cascade,
  streak_date date not null,
  created_at timestamptz not null default now(),
  primary key (user_id, streak_date)
);
create index if not exists idx_user_streaks_user_date on user_streaks(user_id, streak_date desc);

-- Leaderboard: per-user per-course scores
create table if not exists course_scores (
  user_id bigint not null references users(id) on delete cascade,
  course_id bigint not null references courses(id) on delete cascade,
  score int not null default 0,
  completed_at timestamptz,
  primary key (user_id, course_id)
);
create index if not exists idx_course_scores_score on course_scores(score desc);
create index if not exists idx_course_scores_course on course_scores(course_id);

-- Capacity limit for courses (null = unlimited)
alter table courses add column if not exists capacity int;

-- Enrollments: manages the user-course lifecycle
create table if not exists enrollments (
  id bigserial primary key,
  user_id bigint not null references users(id) on delete cascade,
  course_id bigint not null references courses(id) on delete cascade,
  status text not null default 'active'
    check (status in ('active', 'completed', 'dropped', 'pending')),
  progress_pct numeric(5,2) not null default 0,
  enrolled_at timestamptz not null default now(),
  completed_at timestamptz,
  dropped_at timestamptz,
  unique(user_id, course_id)
);
create index if not exists idx_enrollments_user on enrollments(user_id);
create index if not exists idx_enrollments_course on enrollments(course_id);
create index if not exists idx_enrollments_status on enrollments(status);

-- Learning Paths: sequenced curricula grouping multiple courses
create table if not exists learning_paths (
  id bigserial primary key,
  title text not null,
  description text not null default '',
  created_by bigint not null references users(id) on delete cascade,
  status text not null default 'draft'
    check (status in ('draft', 'published', 'archived')),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

-- Courses within a learning path (ordered sequence)
create table if not exists learning_path_courses (
  id bigserial primary key,
  learning_path_id bigint not null references learning_paths(id) on delete cascade,
  course_id bigint not null references courses(id) on delete cascade,
  sort_order int not null default 0,
  is_required boolean not null default true,
  unique(learning_path_id, course_id)
);
create index if not exists idx_lpc_path on learning_path_courses(learning_path_id);
create index if not exists idx_lpc_course on learning_path_courses(course_id);

-- User enrollments in learning paths
create table if not exists learning_path_enrollments (
  id bigserial primary key,
  user_id bigint not null references users(id) on delete cascade,
  learning_path_id bigint not null references learning_paths(id) on delete cascade,
  status text not null default 'active'
    check (status in ('active', 'completed', 'dropped')),
  progress_pct numeric(5,2) not null default 0,
  started_at timestamptz not null default now(),
  completed_at timestamptz,
  dropped_at timestamptz,
  unique(user_id, learning_path_id)
);
create index if not exists idx_lpe_user on learning_path_enrollments(user_id);
create index if not exists idx_lpe_path on learning_path_enrollments(learning_path_id);

-- Certificates: issued when a user completes a course
create table if not exists certificates (
  id bigserial primary key,
  user_id bigint not null references users(id) on delete cascade,
  course_id bigint not null references courses(id) on delete cascade,
  issued_at timestamptz not null default now(),
  certificate_code text not null unique,  -- UUID-based unique code for verification
  score_pct numeric(5,2) not null default 0,
  tier text not null default 'bronze' check (tier in ('bronze', 'silver', 'gold')),
  unique(user_id, course_id)
);
create index if not exists idx_certificates_user on certificates(user_id);
create index if not exists idx_certificates_course on certificates(course_id);
create index if not exists idx_certificates_code on certificates(certificate_code);

-- Badges: milestone achievements that can be earned (not scoped to a single course)
create table if not exists badge_definitions (
  id bigserial primary key,
  code text not null unique,           -- e.g. 'first_course', 'streak_7', 'speed_demon'
  name text not null,
  description text not null default '',
  icon text not null default 'star',   -- lucide icon name
  tier text not null default 'bronze' check (tier in ('bronze', 'silver', 'gold')),
  category text not null default 'achievement' check (category in ('achievement', 'streak', 'milestone', 'speed', 'skill')),
  criteria jsonb not null default '{}', -- e.g. {"type":"courses_completed","count":1}
  created_at timestamptz not null default now()
);

-- Badge awards: which user earned which badge
create table if not exists badge_awards (
  id bigserial primary key,
  user_id bigint not null references users(id) on delete cascade,
  badge_id bigint not null references badge_definitions(id) on delete cascade,
  earned_at timestamptz not null default now(),
  metadata jsonb not null default '{}', -- e.g. {"course_id": 5, "streak_count": 7}
  unique(user_id, badge_id)
);
create index if not exists idx_badge_awards_user on badge_awards(user_id);
create index if not exists idx_badge_awards_badge on badge_awards(badge_id);

-- Seed badge definitions
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

-- Password reset tokens: expiring tokens for forgot-password flow
create table if not exists password_reset_tokens (
  id bigserial primary key,
  user_id bigint not null references users(id) on delete cascade,
  token text not null unique,
  expires_at timestamptz not null,
  used boolean not null default false,
  created_at timestamptz not null default now()
);
create index if not exists idx_password_reset_token on password_reset_tokens(token);
create index if not exists idx_password_reset_user on password_reset_tokens(user_id);
