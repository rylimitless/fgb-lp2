-- name: CheckIfFirstUser :one
select count(*) from users;

-- name: CreateUser :one
insert into users (email, password_hash, name, role)
values ($1, $2, $3, $4)
returning *;


-- name: UpdateCourseItemData :one
update course_items set data = $2 where id = $1 returning *;

-- name: UpdateCourseItemModule :one
update course_items set module_id = $2, sort_order = $3 where id = $1 returning *;

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

-- name: UpdateCourseSettings :one
update courses set settings = $2, updated_at = now() where id = $1 returning *;

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

-- name: GetCourseItemByID :one
select * from course_items where id = $1;

-- name: GetCourseItemsByModule :many
select * from course_items where module_id = $1 order by sort_order asc;

-- name: GetCourseItemsByCourse :many
select * from course_items where course_id = $1 order by sort_order asc;

-- name: DeleteCourseModules :exec
delete from modules where course_id = $1;

-- name: DeleteCourseItems :exec
delete from course_items where course_id = $1;

-- name: DeleteCourseItemByID :exec
delete from course_items where id = $1;

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

-- name: EnrollInCourse :exec
insert into lesson_progress (user_id, course_id, current_module, completed, score_pct)
values ($1, $2, 0, false, 0)
on conflict (user_id, course_id) do nothing;

-- name: ResetLessonProgressStartedAt :exec
update lesson_progress set started_at = now() where user_id = $1 and course_id = $2;

-- name: UpsertItemProgress :one
insert into item_progress (user_id, course_id, item_id, answer, is_correct)
values ($1, $2, $3, $4, $5)
on conflict (user_id, item_id)
do update set answer = $4, is_correct = $5, answered_at = now()
returning *;

-- name: GetItemProgressByCourse :many
select * from item_progress where user_id = $1 and course_id = $2;

-- name: DeleteItemProgress :exec
delete from item_progress where user_id = $1 and course_id = $2;

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

-- name: GetUserRoles :many
select role from user_roles where user_id = $1;

-- name: InsertUserRole :exec
insert into user_roles (user_id, role)
values ($1, $2)
on conflict (user_id, role) do nothing;

-- name: DeleteUserRoles :exec
delete from user_roles where user_id = $1;

-- name: UpdateUserPrimaryRole :exec
update users set role = $2, updated_at = now() where id = $1;

-- Streaks & Leaderboard --

-- name: RecordStreak :exec
insert into user_streaks (user_id, streak_date)
values ($1, current_date)
on conflict (user_id, streak_date) do nothing;

-- name: GetUserStreak :one
with dates as (
  select streak_date,
    streak_date - (row_number() over (order by streak_date desc))::int as grp
  from user_streaks
  where user_id = $1
    and streak_date <= current_date
),
consecutive as (
  select count(*) as cnt
  from dates
  where grp = (
    select grp from dates where streak_date = (select max(streak_date) from dates)
  )
)
select coalesce((select cnt from consecutive), 0) as streak_days;

-- name: UpsertCourseScore :one
insert into course_scores (user_id, course_id, score, completed_at)
values ($1, $2, $3, case when $4 then now() else null end)
on conflict (user_id, course_id)
do update set score = course_scores.score + $3,
  completed_at = case when $4 then now() else course_scores.completed_at end
returning *;

-- name: GetLeaderboard :many
select u.id as user_id, u.name as user_name, coalesce(sum(cs.score), 0)::int as total_score
from users u
left join course_scores cs on cs.user_id = u.id
group by u.id, u.name
order by total_score desc
limit $1;

-- name: GetUserTotalScore :one
select coalesce(sum(score), 0) as total_score
from course_scores
where user_id = $1;

-- name: GetCourseAttemptCount :one
select coalesce(count(*), 0)::int as attempt_count
from lesson_progress
where user_id = $1 and course_id = $2 and completed = true;

-- Departments --

-- name: CreateDepartment :one
insert into departments (name)
values ($1)
returning *;

-- name: GetDepartments :many
select * from departments order by name asc;

-- name: GetDepartmentByID :one
select * from departments where id = $1;

-- name: DeleteDepartment :exec
delete from departments where id = $1;

-- name: AddUserToDepartment :exec
insert into user_departments (user_id, department_id)
values ($1, $2)
on conflict (user_id, department_id) do nothing;

-- name: RemoveUserFromDepartment :exec
delete from user_departments
where user_id = $1 and department_id = $2;

-- name: GetUserDepartments :many
select d.* from departments d
join user_departments ud on ud.department_id = d.id
where ud.user_id = $1
order by d.name asc;

-- name: GetUsersByDepartment :many
select u.id, u.email, u.name, u.role, u.created_at
from users u
join user_departments ud on ud.user_id = u.id
where ud.department_id = $1
order by u.name asc;

-- Enrollments --

-- name: GetUserEnrollments :many
select e.*, c.title as course_title, c.description as course_description, c.status as course_status
from enrollments e
join courses c on c.id = e.course_id
where e.user_id = $1
order by e.enrolled_at desc;

-- name: GetCourseEnrollment :one
select * from enrollments where user_id = $1 and course_id = $2;

-- name: CreateEnrollment :one
insert into enrollments (user_id, course_id, status)
values ($1, $2, 'active')
on conflict (user_id, course_id)
do update set status = 'active', dropped_at = null
returning *;

-- name: UpdateEnrollmentStatus :one
update enrollments
set status = $2,
    completed_at = case when $2 = 'completed' then now() else completed_at end,
    dropped_at = case when $2 = 'dropped' then now() else dropped_at end
where id = $1
returning *;

-- name: UpdateEnrollmentProgress :one
update enrollments
set progress_pct = $2,
    status = case when $2 >= 100 then 'completed' else status end,
    completed_at = case when $2 >= 100 then now() else completed_at end
where id = $1
returning *;

-- name: GetCourseEnrollments :many
select e.*, u.name as user_name, u.email as user_email
from enrollments e
join users u on u.id = e.user_id
where e.course_id = $1
order by e.enrolled_at desc;

-- name: CountActiveEnrollments :one
select count(*) from enrollments where course_id = $1 and status = 'active';

-- name: DeleteEnrollment :exec
delete from enrollments where id = $1;

-- name: EnrollInCourseV2 :one
with enrolled as (
  insert into enrollments (user_id, course_id, status)
  values ($1, $2, 'active')
  on conflict (user_id, course_id) do nothing
  returning *
)
select * from enrolled
union all
select * from enrollments where user_id = $1 and course_id = $2 and not exists (select 1 from enrolled);

-- name: GetEnrollmentByID :one
select e.*, c.title as course_title, u.name as user_name
from enrollments e
join courses c on c.id = e.course_id
join users u on u.id = e.user_id
where e.id = $1;

-- Learning Paths --

-- name: CreateLearningPath :one
insert into learning_paths (title, description, created_by)
values ($1, $2, $3)
returning *;

-- name: GetLearningPaths :many
select * from learning_paths order by updated_at desc;

-- name: GetLearningPathByID :one
select * from learning_paths where id = $1;

-- name: UpdateLearningPath :one
update learning_paths
set title = $2, description = $3, updated_at = now()
where id = $1
returning *;

-- name: UpdateLearningPathStatus :one
update learning_paths
set status = $2, updated_at = now()
where id = $1
returning *;

-- name: DeleteLearningPath :exec
delete from learning_paths where id = $1;

-- name: AddCourseToPath :one
insert into learning_path_courses (learning_path_id, course_id, sort_order, is_required)
values ($1, $2, $3, $4)
on conflict (learning_path_id, course_id) do nothing
returning *;

-- name: RemoveCourseFromPath :exec
delete from learning_path_courses where learning_path_id = $1 and course_id = $2;

-- name: GetPathCourses :many
select lpc.*, c.title as course_title, c.description as course_description, c.status as course_status
from learning_path_courses lpc
join courses c on c.id = lpc.course_id
where lpc.learning_path_id = $1
order by lpc.sort_order asc;

-- name: UpdatePathCourseOrder :one
update learning_path_courses
set sort_order = $3, is_required = $4
where learning_path_id = $1 and course_id = $2
returning *;

-- name: GetPublishedLearningPaths :many
select * from learning_paths where status = 'published' order by updated_at desc;

-- name: GetPublishedLearningPathByID :one
select * from learning_paths where id = $1 and status = 'published';

-- name: EnrollInLearningPath :one
insert into learning_path_enrollments (user_id, learning_path_id, status)
values ($1, $2, 'active')
on conflict (user_id, learning_path_id)
do update set status = 'active', dropped_at = null
returning *;

-- name: GetLearningPathEnrollment :one
select * from learning_path_enrollments
where user_id = $1 and learning_path_id = $2;

-- name: GetUserLearningPathEnrollments :many
select lpe.*, lp.title as path_title, lp.description as path_description
from learning_path_enrollments lpe
join learning_paths lp on lp.id = lpe.learning_path_id
where lpe.user_id = $1
order by lpe.started_at desc;

-- name: UpdateLearningPathEnrollmentProgress :one
update learning_path_enrollments
set progress_pct = $2,
    status = case when $2 >= 100 then 'completed' else status end,
    completed_at = case when $2 >= 100 then now() else completed_at end
where id = $1
returning *;

-- name: DropLearningPathEnrollment :one
update learning_path_enrollments
set status = 'dropped', dropped_at = now()
where id = $1
returning *;

-- name: CountPathCourses :one
select count(*)::int from learning_path_courses where learning_path_id = $1;

-- name: IssueCertificate :one
insert into certificates (user_id, course_id, certificate_code, score_pct, tier)
values ($1, $2, $3, $4, $5)
on conflict (user_id, course_id) do update
  set score_pct = $4, tier = $5, issued_at = now()
returning *;

-- name: GetUserCertificates :many
select c.*, co.title as course_title, co.description as course_description
from certificates c
join courses co on co.id = c.course_id
where c.user_id = $1
order by c.issued_at desc;

-- name: GetCertificateByCode :one
select c.*, co.title as course_title, u.name as user_name
from certificates c
join courses co on co.id = c.course_id
join users u on u.id = c.user_id
where c.certificate_code = $1;

-- name: GetCourseCertificate :one
select * from certificates where user_id = $1 and course_id = $2;

-- name: CountUserCertificates :one
select count(*) from certificates where user_id = $1;

-- name: GetAllBadgeDefinitions :many
select * from badge_definitions order by id;

-- name: GetBadgeDefinitionByCode :one
select * from badge_definitions where code = $1;

-- name: AwardBadge :one
insert into badge_awards (user_id, badge_id, metadata)
values ($1, $2, $3)
on conflict (user_id, badge_id) do nothing
returning *;

-- name: GetUserBadges :many
select ba.*, bd.code as badge_code, bd.name as badge_name, bd.description as badge_description,
       bd.icon as badge_icon, bd.tier as badge_tier, bd.category as badge_category
from badge_awards ba
join badge_definitions bd on bd.id = ba.badge_id
where ba.user_id = $1
order by ba.earned_at desc;

-- name: HasUserBadge :one
select count(*) > 0 as has_badge
from badge_awards ba
join badge_definitions bd on bd.id = ba.badge_id
where ba.user_id = $1 and bd.code = $2;

-- name: CountUserBadges :one
select count(*) from badge_awards where user_id = $1;

-- name: CountCompletedCourses :one
select count(*) from enrollments where user_id = $1 and status = 'completed';
