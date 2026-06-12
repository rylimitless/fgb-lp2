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
