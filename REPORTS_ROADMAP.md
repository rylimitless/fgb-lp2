# Reports Roadmap

Reference for making the `/analytics` reports section reflective of every feature
currently in the platform. Build incrementally — one chunk at a time — checking
off each item as it ships.

## Current state (as of writing)

Reports live at `/analytics` with two tabs: **Platform** (admin/auditor) and
**Team** (manager).

### Platform tab
- 4 stat cards: Active learners (30d), Total users, Coach queries, Adaptive θ.
- Most-failed topics (bar chart).
- Course effectiveness (bar + line chart) — avg score vs. learner count.
- Adaptive radar — θ per topic.
- Coach usage over time (line, last 30d).
- Recent audit log (table, latest 10).

### Team tab (managers)
- 4 stat cards: Departments, Active learners, Struggling learners, Avg completion.
- Team overview table — per department: members, active learners, completed, avg progress.
- Struggling learners table — name, dept, avg score, active courses, last active.
- Completion rates table — per department: enrolled, completed, completion %.

---

## Roadmap (in build order)

### #1 — Enrollment & completion dashboard  🔴 (START HERE)
Biggest gap; ties directly to the enrollment/bulk-upload features already built.
**Status: ✅ DONE**

- [x] Platform-wide enrollment summary: total / active / completed / dropped / pending, completion rate, avg progress, unique enrolled learners.
- [x] Enrollments over time (trend) — new enrollments + completions per day (configurable window).
- [x] Top courses by enrollment — count, completed, completion %.
- [ ] Completion funnel — enrolled → in-progress → completed → dropped (counts shown in summary; dedicated viz pending).
- [x] Stalled/dropped enrollments — started but no recent activity (14+ days).
- [x] Avg time-to-complete per course (enrolled_at → completed_at).

Endpoints: `GET /api/analytics/enrollments/overview`,
`/enrollments/timeline?days=30`, `/enrollments/stalled`.

### #2 — Certificates & badges reporting  🔴
Gamification/certificates are first-class but have zero reporting presence.
**Status: ✅ DONE**

- [x] Certificates issued — count, over time, by tier (bronze/silver/gold).
- [x] Badges awarded — count, by badge, by tier.
- [x] Top earners — most certs / badges (combined leaderboard).
- [x] Certificate tier distribution (gold/silver/bronze shown as stat cards).
- [x] Most-awarded badges table (with category + tier).

Endpoints: `GET /api/analytics/credentials/overview`,
`/credentials/timeline?days=30`.

### #3 — User growth & composition  🟡
**Status: ✅ DONE**

- [x] Signups over time (especially relevant after bulk uploads).
- [x] Users by role (end user / creator / admin / …).
- [x] Users by department (platform-wide distribution).
- [x] Users without department (orphans).
- [x] Bulk-created vs. individual (from audit_log `users_bulk_created`).

Endpoints: `GET /api/analytics/users/overview`, `/users/growth?days=30`.

### #4 — Learning paths reporting  🟡
Learning paths have their own enrollment tracking, parallel to courses.
**Status: ✅ DONE**

- [x] Learning path enrollment & completion.
- [x] Path progress (avg % across enrolled).
- [x] Most popular paths.

Endpoint: `GET /api/analytics/learning-paths/overview`.

### #5 — Content reporting  🟡
**Status: ✅ DONE**

- [x] Courses by status (draft / published / archived).
- [x] Documents by status (uploaded / processing / ready / failed).
- [x] Content authored per creator.
- [x] Review queue health (pending / approved / rejected counts).

Endpoint: `GET /api/analytics/content/overview`.

### #6 — Engagement refinement  🟢
**Status: ✅ DONE**

- [x] Streaks — active streaks (7d), avg streak length, longest streak.
- [x] Gamification — practice session volume + avg score.
- [x] Notifications (unread count). Low priority items surfaced as stat cards.

Endpoint: `GET /api/analytics/engagement/overview`.

### #7 — Cross-cutting capabilities  🟢
**Status: ✅ DONE**

- [x] Date-range filter on all time-series charts (7d / 30d / 90d / 1y) —
      applies to enrollments, credentials, user-growth, and coach-usage timelines.
      `coach-usage` now also accepts a `days` param (was hardcoded to 30).
- [x] CSV export of report tables (top courses, top earners, users-by-department).
- [ ] Per-department drilldown from platform view (deferred).

---

## Notes
- Analytics routes are registered under `auditorManagerGroup` (admin, auditor,
  manager) in `backend/main.go`. Team-specific routes use the same group.
- Platform analytics use sqlc queries (`database.Queries.*`); newer team
  analytics use direct `Pool.Query` — both patterns are acceptable.
- Frontend renders with Chart.js (canvases). Stat cards use the `StatCard`
  brand component.
