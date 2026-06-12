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
    status = case when $4 then 'published' else status end,
    updated_at = now()
where id = $1
returning *;
