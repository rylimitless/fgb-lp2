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
    chunks_done = coalesce(sqlc.narg('chunks_done'), chunks_done)
where id = $1
returning *;

-- name: InsertDocumentChunk :one
insert into document_chunks (document_id, chunk_index, content, embedding)
values ($1, $2, $3, $4)
returning *;
