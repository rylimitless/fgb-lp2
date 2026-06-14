-- name: CheckIfFirstUser :one
select count(*) from users;

-- name: CreateUser :one
insert into users (email, password_hash, name, role)
values ($1, $2, $3, $4)
returning *;

-- name: GetUserByEmail :one
select * from users where email = $1;

-- name: GetUserByID :one
select * from users where id = $1;

-- name: CreateSession :one
insert into sessions (user_id, token, expires_at)
values ($1, $2, now() + interval '24 hours')
returning *;

-- name: GetSessionByToken :one
select * from sessions where token = $1 and expires_at > now();

-- name: DeleteSession :exec
delete from sessions where token = $1;

-- name: GetAllUsers :many
select * from users order by created_at desc;

-- name: DeleteUser :exec
delete from users where id = $1;

-- name: InsertDocument :one
insert into documents (title, file_path, status, uploaded_by)
values ($1, $2, 'uploaded', $3)
returning *;

-- name: GetDocuments :many
select * from documents order by created_at desc;

-- name: GetDocumentByID :one
select * from documents where id = $1;

-- name: DeleteDocument :exec
delete from documents where id = $1;

-- name: GetPendingDocuments :many
select * from documents where status = 'uploaded' order by created_at asc;

-- name: ClaimDocument :one
update documents
set status = 'processing'
where id = $1 and status = 'uploaded'
returning *;

-- name: UpdateDocumentStatus :one
update documents
set status = $2,
    total_chunks = coalesce(sqlc.narg('total_chunks'), total_chunks),
    chunks_done = coalesce(sqlc.narg('chunks_done'), chunks_done),
    error_message = coalesce(sqlc.narg('error_message'), error_message),
    approved = coalesce(sqlc.narg('approved'), approved),
    review_status = coalesce(sqlc.narg('review_status'), review_status)
where id = $1
returning *;

-- name: InsertDocumentChunk :one
insert into document_chunks (document_id, chunk_index, content, embedding)
values ($1, $2, $3, $4)
returning *;

-- name: SearchDocumentChunks :many
select dc.*, d.title as document_title
from document_chunks dc
join documents d on d.id = dc.document_id
where d.approved = true and dc.embedding is not null
order by dc.embedding <=> $1
limit $2;

-- name: GetDocumentChunks :many
select * from document_chunks where document_id = $1 order by chunk_index;

-- name: GetApprovedDocuments :many
select * from documents where approved = true order by created_at desc;

-- name: CreateCourse :one
insert into courses (title, description, created_by, source_doc_ids, settings)
values ($1, $2, $3, $4, $5)
returning *;

-- name: GetCourses :many
select * from courses order by updated_at desc;

-- name: GetCourseByID :one
select * from courses where id = $1;

-- name: UpdateCourseMeta :one
update courses set title = $2, description = $3, updated_at = now() where id = $1 returning *;

-- name: UpdateCourseStatus :one
update courses set status = $2, updated_at = now() where id = $1 returning *;

-- name: DeleteCourse :exec
delete from courses where id = $1;

-- name: CreateModule :one
insert into modules (course_id, title, description, sort_order)
values ($1, $2, $3, $4)
returning *;

-- name: GetModulesByCourse :many
select * from modules where course_id = $1 order by sort_order asc;

-- name: CreateCourseItem :one
insert into course_items (course_id, module_id, item_type, sort_order, data)
values ($1, $2, $3, $4, $5)
returning *;

-- name: GetCourseItemsByModule :many
select * from course_items where module_id = $1 order by sort_order asc;

-- name: GetCourseItemsByCourse :many
select * from course_items where course_id = $1 order by sort_order asc;

-- name: DeleteCourseModules :exec
delete from modules where course_id = $1;

-- name: DeleteCourseItems :exec
delete from course_items where course_id = $1;

-- name: GetPendingReviewDocuments :many
select * from documents where review_status = 'pending' order by created_at desc limit $1 offset $2;

-- name: CountPendingReviewDocuments :one
select count(*) from documents where review_status = 'pending';

-- name: GetPendingReviewCourses :many
select * from courses where review_status = 'pending' order by updated_at desc limit $1 offset $2;

-- name: CountPendingReviewCourses :one
select count(*) from courses where review_status = 'pending';

-- name: GetPublishedCourses :many
select * from courses where status = 'published' and approved = true order by updated_at desc;

-- name: SearchPublishedCourses :many
select *, ts_rank(to_tsvector('english', title || ' ' || description), plainto_tsquery('english', $1)) as rank
from courses
where status = 'published' and approved = true
  and to_tsvector('english', title || ' ' || description) @@ plainto_tsquery('english', $1)
order by rank desc
limit $2;

-- name: GetAllLessonProgress :many
select * from lesson_progress where user_id = $1;

-- name: UpsertLessonProgress :one
insert into lesson_progress (user_id, course_id, current_module, completed, score_pct)
values ($1, $2, $3, $4, $5)
on conflict (user_id, course_id)
do update set current_module = $3, completed = $4, score_pct = $5, completed_at = case when $4 then now() else lesson_progress.completed_at end
returning *;

-- name: GetLessonProgress :one
select * from lesson_progress where user_id = $1 and course_id = $2;

-- name: GetLearningPreference :one
select * from learning_preferences where user_id = $1;

-- name: UpsertLearningPreference :one
insert into learning_preferences (user_id, learning_style, difficulty_level, theta)
values ($1, 'mixed', 'adaptive', $2)
on conflict (user_id)
do update set theta = $2, difficulty_level = 'adaptive', updated_at = now()
returning *;

-- name: UpdateDocumentReview :one
update documents
set review_status = $2,
    review_notes = $3,
    approved = $4
where id = $1
returning *;

-- name: UpdateCourseReview :one
update courses
set review_status = $2,
    review_notes = $3,
    approved = $4,
    approved_by = $5,
    status = case when $4 then 'published' else status end,
    updated_at = now()
where id = $1
returning *;

-- name: GetUserNotifications :many
select * from notifications where user_id = $1 order by created_at desc limit $2;

-- name: CountUnreadNotifications :one
select count(*) from notifications where user_id = $1 and is_read = false;

-- name: CreateNotification :one
insert into notifications (user_id, title, message, link)
values ($1, $2, $3, $4)
returning *;

-- name: MarkNotificationRead :one
update notifications set is_read = true where id = $1 returning *;

-- name: GetAdminUsers :many
select * from users where role = 'admin' order by created_at;

-- name: InsertCoachQuery :one
insert into coach_queries (user_id, question, sources_count)
values ($1, $2, $3)
returning *;

-- name: CountCoachQueries :one
select count(*) from coach_queries;

-- name: CountActiveLearners :one
select count(distinct user_id) from (
  select user_id from practice_sessions where started_at > now() - interval '30 days'
  union
  select user_id from lesson_progress where started_at > now() - interval '30 days'
) sub;

-- name: CountTotalUsers :one
select count(*) from users;

-- name: GetMostFailedTopics :many
select
  topic,
  sum(question_count)::int as total_questions,
  sum(correct_count)::int as total_correct,
  sum(question_count - correct_count)::int as total_wrong
from practice_sessions
where completed_at is not null
group by topic
order by total_wrong desc
limit $1;

-- name: GetCourseEffectiveness :many
select
  c.id as course_id,
  c.title as course_title,
  count(lp.user_id) as learner_count,
  coalesce(avg(lp.score_pct), 0) as avg_score,
  count(case when lp.completed then 1 end) as completed_count
from courses c
left join lesson_progress lp on lp.course_id = c.id
where c.status = 'published' and c.approved = true
group by c.id, c.title
order by avg_score desc;

-- name: GetAdaptiveOverview :one
select
  count(distinct user_id) as active_users,
  coalesce(avg(theta), 0) as avg_theta,
  coalesce(stddev(theta), 0) as stddev_theta,
  coalesce(min(theta), 0) as min_theta,
  coalesce(max(theta), 0) as max_theta
from learning_preferences
where updated_at > now() - interval '30 days';

-- name: GetCoachQueriesOverTime :many
select
  date_trunc('day', created_at)::date as day,
  count(*)::int as query_count
from coach_queries
group by day
order by day desc
limit $1;

-- name: InsertAuditLog :one
insert into audit_log (user_id, action, details)
values ($1, $2, $3)
returning *;

-- name: GetAuditLogs :many
select
  al.*,
  u.name as user_name,
  u.email as user_email,
  u.role as user_role
from audit_log al
left join users u on u.id = al.user_id
order by al.created_at desc
limit $1 offset $2;
