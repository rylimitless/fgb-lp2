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
  request jsonb not null,
  steps jsonb not null default '[]',
  modules jsonb not null default '[]',
  result jsonb,
  error text,
  course_id bigint references courses(id) on delete set null,
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
