# Audit Status — What's Fixed and What's Still Missing

Cross-references `AUDIT.md` against the current codebase. Each item is marked
**FIXED**, **PARTIAL**, or **NOT FIXED** with a short verification note.

Last verified: 2026-08-03 (one-liner pass applied).

> **2026-08-03 update:** A focused "one-liner fixes" pass has been applied.
> Items now resolved are marked ✅ in the tables below; see the changelog at
> the bottom of this file. Go backend passes `go build ./...` and
> `go vet ./...`. Frontend `npm run check` could not be run in this
> environment due to a pre-existing broken `rolldown` native binding
> (unrelated to these changes).

---

## Summary counts

| Severity | Total | Fixed | Partial | Not Fixed |
|---|---|---|---|---|
| 🔴 Critical | 6 | 2 | 0 | 4 |
| 🟠 High | 14 | 13 | 1 | 0 |
| 🟡 Medium | 30 | 9 | 1 | 20 |
| 🟢 Low | ~18 | 0 | – | ~18 |
| 📦 Missing features | 18 | 0 | 1 | 17 |

**Bottom line:** 2 of 6 Criticals done (C2, C5). **13 of 14 Highs done** (H11
partial — timeout done, retry/DLQ/UI pending). Remaining Criticals (C1, C3, C4,
C6) are now the priority.

---

## 🔴 Critical — all 6 still open

| # | Status | Verification |
|---|---|---|
| **C1** Password hashing broken | ❌ NOT FIXED | `backend/functions/hash_functions.go:19-25` still uses static salt `"testsalt"` + `19*1024` bytes memory. No argon2id package, no encoded params, no re-hash-on-login migration. |
| **C2** Password-reset Origin poisoning | ✅ FIXED | `backend/password_reset/handler.go` now reads `os.Getenv("APP_URL")` (falls back to the hardcoded prod URL) instead of the client `Origin` header. The misleading comment is gone too. |
| **C3** Path traversal in uploads | ❌ NOT FIXED | `backend/worker/worker.go:339-352` `GenerateFilePath` still uses the broken `strings.Map` that only strips spaces/brackets — `/` and `..` pass through. |
| **C4** Hardcoded DB credentials | ❌ NOT FIXED | `compose.yaml:7` and `compose-production.yaml:9` both still `POSTGRES_PASSWORD: password`. |
| **C5** No pgvector index | ✅ FIXED | Added `CREATE INDEX IF NOT EXISTS document_chunks_embedding_hnsw ON document_chunks USING hnsw (embedding vector_cosine_ops)` in both `schema.sql` (fresh installs) and `runMigrations` (existing installs). Will apply on next `docker compose up`. |
| **C6** Prod Dockerfile runs as root, not multi-stage | ❌ NOT FIXED | `backend/Dockerfile.prod` still single-stage `golang:1.26-bookworm`, builds then `CMD ["go","run","main.go"]`. `compose-production.yaml:54-56` still bind-mounts `./backend:/app`. |

---

## 🟠 High

### Fixed ✅

| # | Verification |
|---|---|
| **H1** IDOR on learning-path enrollments | FIXED. `backend/learning_paths/handler.go:949` `DropEnrollment` now verifies ownership before mutating ("IDOR-safe: requires the enrollment to belong to the requesting user"). `UpdateProgress` was removed in favor of server-computed `RecomputePathProgress`. |
| **H13** Chart.js leak on `/analytics` | FIXED. `src/routes/(app)/analytics/+page.svelte:196` `activeCharts.forEach((c) => c.destroy())` in `$effect` cleanup. |
| **H14** Polling `$effect` stacks intervals | FIXED. `src/routes/(app)/content-studio/+page.svelte:30-42` now clears the previous interval and returns a cleanup function. |
| **H8** Frontend has no server-side role guards | FIXED. `src/hooks.server.ts` now resolves the user once per request (via `/api/me`), stores it on `event.locals.user`, and enforces a route-prefix → roles map that mirrors the sidebar nav config. Authenticated users lacking the required role are redirected to `/` before the page loads. Routes gated: `/content-studio`, `/course-builder`, `/ai-content-generator` (content creator); `/learning-path-management` (admin/manager/content creator); `/review-queue` (approver); `/analytics` (admin/manager/auditor); `/user-management`, `/audit-log` (admin/auditor); `/department-management` (admin/manager). Admin bypasses all checks. `app.d.ts` `App.Locals` now typed. `+layout.server.ts` reuses `locals.user` (no more duplicate `/api/me` call). |
| **H2** IDOR on notifications | FIXED. `backend/notifications/handler.go` `MarkRead` now runs `UPDATE notifications SET is_read=true WHERE id=$1 AND user_id=$2` (user-scoped) and returns 404 when no row is touched — so a learner can't suppress another user's notification by guessing its id. (The unscoped `MarkNotificationRead` sqlc query is now unused.) |
| **H9** `SaveProgress` trusts client `completed`+`score_pct` | FIXED. `backend/lessons/handler.go` `SaveProgress` now ignores the client `score_pct` and computes the score server-side via `computeCourseScore`: correct `item_progress` rows / assessable `course_items` (any `item_type <> 'content'`). That server-computed score drives lesson_progress, enrollment progress, certificate tier, and the completion email. `Completed` is still accepted as the learner's "I'm done" trigger, but the score — which determines gold/silver/bronze — can no longer be self-reported. (Note: per-item `is_correct` is still client-reported in `SaveItemProgress`; see open note below.) |
| **H5** Login timing oracle | FIXED. `backend/app/auth.go` `Login` now runs a dummy `ValidateHash` against a precomputed hash on the user-not-found path, so not-found and invalid-password take roughly the same time (no email enumeration via timing). |
| **H6** `/api/setup` TOCTOU | FIXED. `backend/app/auth.go` `SetupAdmin` now runs the first-user check + create inside a single transaction guarded by `pg_advisory_xact_lock(hashtext('setup_admin'))`, so two concurrent setup requests serialize and can't both create an admin. (The route remains registered; the advisory lock + atomic check make that safe.) |
| **H10** Enrollment capacity race | FIXED. `backend/enrollments/handler.go` `EnrollSelf` now does the capacity check + insert atomically: when a course has a capacity, it opens a tx, `SELECT ... FOR UPDATE` on the course row, re-counts active enrollments, and inserts inside the same tx. Added `Pool`/`WithPool` to the handler and wired it in `main.go`. |
| **H3** No rate limiting | FIXED. New dependency-free in-memory token-bucket limiter (`backend/middlewares/rate_limit.go`) applied to the brute-force/email-bombing surfaces: `/api/login` (10 burst / 5 per min), `/api/setup` (5 burst / 1 per min), `/api/forgot-password`+`/reset-password` (3 burst / 1 per min via a route group), and the LLM-costly AI routes `/courses/generate`, `/discover`, `/outline`, `/outline/regenerate` (10 burst / 5 per min). Returns 429 when exceeded. Single-instance (one backend container); swap for Redis limiter if horizontally scaled. |
| **H12** Stale `*gin.Context` in goroutines | FIXED. Swept every `go func()` that fires mailers / audit logs: all capture scalar values and pass `nil` to `audit.Logf` + `context.Background()` — none read the pooled gin context after the handler returns. The coach handler already extracted the user id into a scalar before its goroutine. |
| (bonus) Coach goroutine reads scalar | `backend/coach/handler.go:85-92` extracts `uid := getUserID(c)` before the goroutine — addresses the pooled-context part of **H12**. |

### Partial 🟡

| # | Status |
|---|---|
| **H11** Workers have no retry / DLQ / timeout | PARTIAL — **timeout done, retry/DLQ/admin-UI not done.** `backend/worker/worker.go` now wraps each document in `context.WithTimeout(..., 15*time.Minute)`, and the embeddings client gained an `EmbedCtx` variant used by the worker so a hung embeddings service is interrupted by the deadline (previously blocked up to the HTTP client's 60s). Still missing: `attempts`/`last_error` columns + transient-error retry/backoff on `documents` and `course_generation_jobs`, and an admin UI to re-queue failed jobs. |

### Not fixed ❌

| # | Verification |
|---|---|
| _(no remaining unfixed Highs — see Partial above for H11)_ | |

---

## 🟡 Medium

### Fixed in this pass ✅

| # | Verification |
|---|---|
| **M5** `ApproveDocument` always sets `status='ready'` | FIXED. `backend/documents/handler.go` now sets `status = "rejected"` when `approved == false`. Schema constraint widened to allow `'rejected'` (both `schema.sql` for fresh installs and a `runMigrations` drop/recreate for existing DBs). |
| **M6** `IssueForCourse` defaults score to 100 | FIXED. `backend/certificates/handler.go` now defaults `scorePct := 0.0` instead of 100. **Behavior note:** a completion with no `course_scores` row now yields a 0%-tier (bronze/no) certificate instead of gold. Legitimate completions write a `course_scores` row via `ScoreCourse` before this runs, so real users are unaffected; only the previously-fraudulent path changes. |
| **M19** `permissions` seed lacks `ON CONFLICT DO NOTHING` | FIXED. `backend/database/schema.sql` permissions INSERT now ends with `ON CONFLICT (name) DO NOTHING`. |
| **M21** CORS allowlist contains personal ngrok URL | FIXED. Removed `98fe-173-225-243-241.ngrok-free.app` from both `backend/main.go` CORS allowlist and `vite.config.ts` `server.allowedHosts`. |
| **M22** Embeddings healthcheck YAML quoting bug (dev compose) | FIXED. `compose.yaml` embeddings healthcheck Python snippet now wrapped in double quotes, matching the production file. |
| **M28** Production backend lacks `restart: unless-stopped` | FIXED. `compose-production.yaml` backend service now has `restart: unless-stopped`. |
| **M29** No backend healthcheck in compose | FIXED. Both `compose.yaml` and `compose-production.yaml` backend services now have a `healthcheck` hitting `/api/health` via `wget`. `wget` added to `backend/Dockerfile.prod` (bookworm doesn't ship it). Dev image is alpine (busybox `wget` already present). |

### Fixed earlier ✅

| # | Verification |
|---|---|
| **M9** Bulk operations have no transaction (users side) | PARTIAL→mostly fixed for users: `UpdateUserRoles` (`backend/users/handler.go:198`) and `createUsersFromRows` (`:472`) both use `Pool.Begin`/`Commit`. **But `BulkEnrollUsers`** (`backend/enrollments/handler.go:417-499`) still does per-row inserts in a loop with no transaction. |
| **M23** Svelte 5 `$effect` misused for one-shot side effects | FIXED in `src/routes/(app)/gia-coach/+page.svelte` (now uses `onMount`). Worth a sweep of the other listed spots. |

### Partial 🟡

| # | Status |
|---|---|
| **M27** `runMigrations` no-op | PARTIAL. `backend/main.go` now runs real `ALTER TABLE ... IF NOT EXISTS` statements (and now also applies the C5 HNSW index + M5 constraint). Still not a real migration tool — no versioning, no rollback, no `golang-migrate`/`atlas`. |

### Not fixed ❌

| # | Verification |
|---|---|
| **M1** Login audit `user_id = NULL` | `backend/app/login.go:44` still calls `audit.Log(a.Queries, nil, …)` with nil context. |
| **M2** Deadline notifications re-created every load | Not deduped in `notifications/handler.go` / `home_handler/dashboard.go`. |
| **M3** Dashboard fires 7+ queries per weekday loop | Still a loop in `home_handler/dashboard.go:319-336`. |
| **M4** No pagination on list endpoints | Still unbounded. |
| **M7** Raw SQL errors leaked to clients | Still common across learning_paths, analytics, gamification handlers. |
| **M8** Mailer goroutines no panic recovery / timeout | `backend/password_reset/handler.go:70-77` still `context.Background()` with no timeout and no recover. |
| **M10** No request body size limit | No `MaxBytesReader` / `MaxMultipartMemory` on uploads or CSV. |
| **M11** No `+error.svelte` | Confirmed — no `+error.svelte` files anywhere in `src/routes/`. |
| **M12** Notification polling fires when tab hidden | `src/routes/(app)/+layout.svelte:305` still `setInterval(fetchUnreadCount, 30000)` with no `visibilitychange` guard. |
| **M13** Caddy commented out in prod compose | `compose-production.yaml:80-90` still commented. |
| **M14** Embeddings runs Flask dev server | `backend/embeddings/app.py` still `app.run`. |
| **M15** AI edit endpoints skip review flow | Not enforced in `backend/ai/handler.go` edit endpoints. |
| **M16** Missing FK indexes | `schema.sql` still lacks indexes on `sessions.user_id`, `documents.uploaded_by`, `course_items.course_id`/`module_id`, `modules.course_id`, `practice_sessions.user_id`/`started_at`, `audit_log(action, created_at)`. |
| **M17** `documents`/`course_items`/`modules` lack `updated_at` | Still absent in `schema.sql:64-79, 137-155`. |
| **M18** Hard `ON DELETE CASCADE` everywhere, no soft-delete | No `deleted_at` column anywhere. |
| **M20** `SearchPublishedCourses` no GIN index on tsvector | No `to_tsvector` generated column or GIN index in schema. |
| **M24** Missing `{#each}` keys | Not swept. |
| **M25** `gia-coach`/`adaptive-room` lack `AbortController` | Not added. |
| **M26** Forms lack client-side validation | Not added. |
| **M30** No backup strategy for `pgdata` | None added. |

---

## 🟢 Low / hygiene — none of these have been swept

All still present, including:
- Dead code: `backend/auth_module/hashing.go` (duplicate broken hash), `backend/home_handler/login.go` (empty), unused `totalScore` block in `certificates/handler.go`.
- `app.d.ts` `App.Locals` commented out.
- `<button onclick={() => goto(...)}>` patterns.
- A11y gaps (chat `aria-live`, focus rings, icon-button labels).
- `/test` route still shipped.
- Duplicate `ALTER TABLE learning_preferences ADD COLUMN theta`.
- `scripts/01_certificates_badges_v1.sql` wrong DB name + duplication.
- `vite.config.ts` proxy hardcoded.
- `EXPOSE 5713` typo in `Dockerfile.frontend:10`.
- `course_generation_jobs.id` is `text` PK.
- `users.role` CHECK vs `user_roles.role` drift.
- No structured logging / request IDs.
- Embedding `float64` round-trip.
- `CommandPalette` debounce not cleared.
- Dead `_ = enrolledAtValid` pattern.
- `ListMyEnrollments` N+1 on `settings`.
- `departments.BulkEnroll` inconsistency.
- `UpdateUserRoles` allows self-demotion / last-admin lockout.
- Logout CSRF.

---

## 📦 Missing features — almost entirely still missing

| Capability | Status |
|---|---|
| Migration tool | 🟡 PARTIAL — `runMigrations` is no longer a no-op but no versioned migrations, no rollback. |
| Soft deletes | ❌ Not started. |
| Rate limiting | ❌ Not started. |
| Observability (slog / metrics / traces / request IDs) | ❌ Still `log.Printf` only. |
| CI/CD (`.github/workflows/`) | ❌ No workflows directory exists. |
| Backups for `pgdata` | ❌ None. |
| Security headers (HSTS, CSP, X-Frame-Options, …) | ❌ No middleware. |
| Password change endpoint (`POST /api/me/password`) | ❌ None. |
| Email verification | ❌ No `email_verified` column. |
| MFA / 2FA | ❌ None. |
| Session management UI / rotate-on-login | ❌ None. |
| Graceful shutdown | ❌ `backend/main.go:271` still `r.Run(":5555")`. No `http.Server`, no SIGTERM. |
| Idempotency keys | ❌ None. |
| Frontend i18n | ❌ Not started. |
| Frontend PWA / offline | ❌ No service worker / manifest. |
| File upload progress | ❌ Still plain `fetch`. |
| Dead-letter queue for failed jobs | ❌ None. |
| Frontend pagination UI on all list pages | ❌ Only `content-repository` and `review-queue` paginate. |
| Frontend retry / request cancellation | ❌ No standard API client wrapper. |

---

## Recommended next moves

The High list is essentially closed (13/14 fixed, H11 partial). What remains,
in priority order:

1. **C1** — replace the broken static-salt Argon2id with `alexedwards/argon2id`
   + per-user random salt + re-hash-on-login migration. The biggest remaining
   "exploitable by anyone" item.
2. **C3** — path traversal in `GenerateFilePath`. Fix is small
   (`filepath.Base` + reject `..`), not a deep change.
3. **C4** — rotate the DB password out of version control into env/secrets.
4. **C6** — production Dockerfile (multi-stage, non-root, no source bind-mount).
5. **H11 (finish)** — add `attempts`/`last_error` columns + transient-error
   retry/backoff on documents & course_generation_jobs, and an admin re-queue UI.
6. The remaining Mediums and foundational gaps (real migrations, observability,
   CI, backups).

### Done ✅

- ~~**C5**: one-line `CREATE INDEX` migration.~~
- ~~**C2**: read `APP_URL` from env.~~
- ~~**H4**: `SECURE_COOKIE` env flag + `SameSite: Lax`.~~
- ~~**H7**: `MakeTokens() (string, error)`, fail closed.~~
- ~~**M5, M6, M19, M21, M22, M28, M29**~~
- ~~**H8 frontend**: route→roles map in `hooks.server.ts`.~~
- ~~**H2**: notification IDOR — user-scoped mark-read.~~
- ~~**H9**: `SaveProgress` — server-computed score, client `score_pct` ignored.~~
- ~~**H5**: login timing oracle — dummy hash on not-found.~~
- ~~**H6**: setup TOCTOU — advisory-lock transaction.~~
- ~~**H10**: enrollment capacity race — atomic check+insert.~~
- ~~**H3**: rate limiting on login/setup/forgot-password/AI routes.~~
- ~~**H12**: stale `*gin.Context` in goroutines — swept, all safe.~~
- **H11** (partial): worker per-document timeout + context-aware embeddings. Retry/DLQ/UI still pending.

---

## Changelog — 2026-08-03 one-liner pass

Applied fixes (9 items across 12 files):

| Item | File(s) | Change |
|---|---|---|
| **C2** | `backend/password_reset/handler.go` | Read `APP_URL` from env instead of the client `Origin` header (prevents reset-link poisoning). Added `os` import. |
| **C5** | `backend/database/schema.sql`, `backend/main.go` (`runMigrations`) | Added HNSW index on `document_chunks.embedding` for both fresh installs and existing DBs. |
| **H4** | `backend/main.go` | `setSessionCookie` now honors `SECURE_COOKIE=true` and sets `SameSite=Lax`. |
| **H7** | `backend/functions/hash_functions.go`, `backend/app/auth.go`, `backend/password_reset/handler.go`, `backend/users/handler.go` | `MakeTokens()` now returns `(string, error)` and all 4 callers fail closed on entropy failure. |
| **M5** | `backend/documents/handler.go`, `backend/database/schema.sql`, `backend/main.go` | Rejected documents now get `status='rejected'` instead of `'ready'`; constraint widened to allow it. |
| **M6** | `backend/certificates/handler.go` | Default `scorePct` changed from `100` to `0` (no more unearned gold on missing score rows). |
| **M19** | `backend/database/schema.sql` | Permissions seed now `ON CONFLICT (name) DO NOTHING`. |
| **M21** | `backend/main.go`, `vite.config.ts` | Removed the personal ngrok URL from CORS allowlist and `server.allowedHosts`. |
| **M22** | `compose.yaml` | Quoted the embeddings healthcheck Python snippet (YAML was failing to parse the bare `import ...`). |
| **M28** | `compose-production.yaml` | Backend service now has `restart: unless-stopped`. |
| **M29** | `compose.yaml`, `compose-production.yaml`, `backend/Dockerfile.prod` | Added `/api/health` healthcheck for backend in both compose files; added `wget` to the prod image. |
| **H8** | `src/hooks.server.ts`, `src/app.d.ts`, `src/routes/(app)/+layout.server.ts` | Centralized server-side role guard: hooks resolves the user once, enforces a route-prefix → roles map (mirrors the nav config), and redirects unauthorized users to `/`. `App.Locals.user` typed; layout reuses it (no duplicate `/api/me`). |
| **H2** | `backend/notifications/handler.go` | `MarkRead` now scopes the update with `user_id` and returns 404 on no-op (was an unscoped update letting any user read-suppress any notification). |
| **H9** | `backend/lessons/handler.go`, `backend/main.go` | `SaveProgress` now ignores client `score_pct` and computes the score server-side from `item_progress` joined to assessable `course_items`. Added `computeCourseScore` helper + `Pool`/`WithPool` wiring. The client `completed` flag still acts as the finish trigger, but certificate tier / score can't be self-reported. |
| **H5** | `backend/app/auth.go` | `Login` runs a dummy hash validation on user-not-found to equalize timing (anti-enumeration). |
| **H6** | `backend/app/auth.go` | `SetupAdmin` wraps first-user check + create in a tx with `pg_advisory_xact_lock`; no more TOCTOU double-admin. |
| **H10** | `backend/enrollments/handler.go`, `backend/main.go` | `EnrollSelf` does capacity check + insert atomically in a tx with `SELECT ... FOR UPDATE` on the course row. Added `Pool`/`WithPool`. |
| **H3** | `backend/middlewares/rate_limit.go`, `backend/main.go`, `backend/ai/handler.go` | New in-memory token-bucket limiter; applied to `/api/login`, `/api/setup`, password-reset group, and AI generation routes. 429 on excess. No new deps. |
| **H12** | _(sweep, no edits)_ | Verified all mailer/audit goroutines capture scalar values + `nil` context; none read the pooled gin context post-handler. |
| **H11** (partial) | `backend/worker/worker.go`, `backend/embeddings/client.go` | Per-document `context.WithTimeout(15m)`; embeddings client gained `EmbedCtx` (context-aware) so the deadline actually interrupts hung calls. Retry/DLQ/admin-UI still pending. |

### Validation

- `go build ./...` — ✅ passes.
- `go vet ./...` — ✅ passes.
- TypeScript diagnostics on the 3 changed frontend files (`hooks.server.ts`,
  `app.d.ts`, `+layout.server.ts`) — ✅ clean (zero errors/warnings).
- `npm run check` — ⚠️ could not run end-to-end in this environment due to a
  pre-existing broken `rolldown` native binding in `node_modules`
  (`Cannot find module '../rolldown-binding.linux-x64-gnu.node'`, which also
  breaks `vite-plugin-svelte` config resolution). Unrelated to these changes;
  worth running locally.

### Operational notes

- The C5 index and M5 constraint changes apply automatically on next
  `docker compose up` via `runMigrations`. For a large `document_chunks` table
  the HNSW index build may take a few seconds on first run.
- **Set `SECURE_COOKIE=true` in production** env, otherwise the cookie stays
  non-secure (the flag defaults to false for local HTTP dev).
- **Set `APP_URL`** in production env (e.g. `https://fgbacademy.rybuildstuff.dev`)
  so password-reset links point at the real host.
