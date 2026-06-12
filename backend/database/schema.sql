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


create table document_chunks (
  id bigserial primary key,
  document_id bigint not null references documents(id) on delete cascade,

  chunk_index int not null,
  content text not null,

  page_number int,
  source_label text,

  embedding vector(4096), -- depends on embedding model

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



create table course_items (
  id bigserial primary key,
  course_id bigint not null references courses(id) on delete cascade,
  item_type text not null
    check (item_type in ('mc', 'sa', 'tf', 'fb', 'dd', 'essay')),
  sort_order int not null default 0,
  data jsonb not null default '{}',
  created_at timestamptz not null default now()
);

-- Adaptive Practice Room: stores per-user learning preferences
create table learning_preferences (
  user_id bigint primary key references users(id) on delete cascade,
  learning_style text not null default 'mixed'
    check (learning_style in ('visual', 'textual', 'interactive', 'mixed')),
  difficulty_level text not null default 'intermediate'
    check (difficulty_level in ('beginner', 'intermediate', 'advanced', 'adaptive')),
  preferred_topics text[] default '{}',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

-- Adaptive Practice Room: tracks practice sessions
create table practice_sessions (
  id bigserial primary key,
  user_id bigint not null references users(id) on delete cascade,
  topic text not null default '',
  question_count int not null default 0,
  correct_count int not null default 0,
  score_pct numeric(5,2) not null default 0,
  started_at timestamptz not null default now(),
  completed_at timestamptz
);
