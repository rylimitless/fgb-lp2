package analytics

import (
	"encoding/json"
	database "fgb-lp/database/queries"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	Queries *database.Queries
	Pool    *pgxpool.Pool
}

func NewHandler(queries *database.Queries, pool *pgxpool.Pool) *Handler {
	return &Handler{Queries: queries, Pool: pool}
}

func (h *Handler) GetOverview(c *gin.Context) {
	activeLearners, _ := h.Queries.CountActiveLearners(c.Request.Context())
	totalUsers, _ := h.Queries.CountTotalUsers(c.Request.Context())
	coachQueries, _ := h.Queries.CountCoachQueries(c.Request.Context())

	adaptiveOverview, _ := h.Queries.GetAdaptiveOverview(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{
		"active_learners": activeLearners,
		"total_users":     totalUsers,
		"coach_queries":   coachQueries,
		"adaptive":        adaptiveOverview,
	})
}

func (h *Handler) GetMostFailedTopics(c *gin.Context) {
	topics, err := h.Queries.GetMostFailedTopics(c.Request.Context(), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if topics == nil {
		topics = []database.GetMostFailedTopicsRow{}
	}
	c.JSON(http.StatusOK, topics)
}

func (h *Handler) GetCourseEffectiveness(c *gin.Context) {
	courses, err := h.Queries.GetCourseEffectiveness(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if courses == nil {
		courses = []database.GetCourseEffectivenessRow{}
	}
	c.JSON(http.StatusOK, courses)
}

func (h *Handler) GetCoachUsageOverTime(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365
	}
	data, err := h.Queries.GetCoachQueriesOverTime(c.Request.Context(), int32(days))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if data == nil {
		data = []database.GetCoachQueriesOverTimeRow{}
	}
	c.JSON(http.StatusOK, data)
}

func (h *Handler) GetAuditLogs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	action := c.Query("action") // optional filter

	if limit > 200 {
		limit = 200
	}

	rows, err := h.Queries.GetAuditLogs(c.Request.Context(), database.GetAuditLogsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if rows == nil {
		rows = []database.GetAuditLogsRow{}
	}

	// Filter by action if specified (since GetAuditLogs doesn't support action filter in sqlc query)
	type AuditEntry struct {
		ID        int64  `json:"id"`
		UserID    *int64 `json:"user_id"`
		Action    string `json:"action"`
		Details   any    `json:"details"`
		CreatedAt string `json:"created_at"`
		UserName  string `json:"user_name"`
		UserEmail string `json:"user_email"`
		UserRole  string `json:"user_role"`
	}

	entries := make([]AuditEntry, 0, len(rows))
	for _, row := range rows {
		if action != "" && row.Action != action {
			continue
		}
		entry := AuditEntry{
			ID:        row.ID,
			Action:    row.Action,
			CreatedAt: row.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		}
		if row.UserID.Valid {
			entry.UserID = &row.UserID.Int64
		}
		if row.UserName.Valid {
			entry.UserName = row.UserName.String
		}
		if row.UserEmail.Valid {
			entry.UserEmail = row.UserEmail.String
		}
		if row.UserRole.Valid {
			entry.UserRole = row.UserRole.String
		}
		// Parse details JSON
		if len(row.Details) > 0 {
			var d any
			if err := json.Unmarshal(row.Details, &d); err == nil {
				entry.Details = d
			}
		}
		if entry.Details == nil {
			entry.Details = gin.H{}
		}
		entries = append(entries, entry)
	}

	c.JSON(http.StatusOK, entries)
}

// --- Manager-specific team analytics ---

// teamOverviewRow is returned by the team overview query.
type teamOverviewRow struct {
	DepartmentName   string  `json:"department_name"`
	TotalMembers     int     `json:"total_members"`
	ActiveLearners   int     `json:"active_learners"`
	CompletedCourses int     `json:"completed_courses"`
	AvgProgress      float64 `json:"avg_progress"`
}

// GetTeamOverview returns department-level learning stats for managers.
func (h *Handler) GetTeamOverview(c *gin.Context) {
	rows, err := h.Pool.Query(c.Request.Context(), `
		select
			coalesce(d.name, 'Unassigned') as department_name,
			count(distinct u.id)::int as total_members,
			count(distinct case when lp.started_at > now() - interval '30 days' then u.id end)::int as active_learners,
			count(distinct case when e.status = 'completed' then e.id end)::int as completed_courses,
			coalesce(avg(e.progress_pct), 0) as avg_progress
		from users u
		left join user_departments ud on ud.user_id = u.id
		left join departments d on d.id = ud.department_id
		left join enrollments e on e.user_id = u.id and e.status = 'active'
		left join lesson_progress lp on lp.user_id = u.id
		group by d.name
		order by total_members desc
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var results []teamOverviewRow
	for rows.Next() {
		var r teamOverviewRow
		if err := rows.Scan(&r.DepartmentName, &r.TotalMembers, &r.ActiveLearners, &r.CompletedCourses, &r.AvgProgress); err != nil {
			continue
		}
		results = append(results, r)
	}
	if results == nil {
		results = []teamOverviewRow{}
	}
	c.JSON(http.StatusOK, results)
}

// strugglingLearnerRow represents a learner who may need attention.
type strugglingLearnerRow struct {
	UserID        int64   `json:"user_id"`
	UserName      string  `json:"user_name"`
	UserEmail     string  `json:"user_email"`
	Department    string  `json:"department"`
	AvgScore      float64 `json:"avg_score"`
	ActiveCourses int     `json:"active_courses"`
	LastActive    string  `json:"last_active"`
}

// GetStrugglingLearners identifies users with low average scores or stalled progress.
func (h *Handler) GetStrugglingLearners(c *gin.Context) {
	threshold, _ := strconv.ParseFloat(c.DefaultQuery("threshold", "50"), 64)
	if threshold <= 0 {
		threshold = 50
	}

	rows, err := h.Pool.Query(c.Request.Context(), `
		select
			u.id as user_id,
			u.name as user_name,
			u.email as user_email,
			coalesce(d.name, 'Unassigned') as department,
			coalesce(avg(lp.score_pct), 0) as avg_score,
			count(distinct case when e.status = 'active' then e.id end)::int as active_courses,
			coalesce(to_char(max(lp.started_at), 'YYYY-MM-DD'), 'never') as last_active
		from users u
		left join user_departments ud on ud.user_id = u.id
		left join departments d on d.id = ud.department_id
		left join lesson_progress lp on lp.user_id = u.id
		left join enrollments e on e.user_id = u.id
		where u.role = 'end user'
		   or exists (select 1 from user_roles ur where ur.user_id = u.id and ur.role = 'end user')
		group by u.id, u.name, u.email, d.name
		having coalesce(avg(lp.score_pct), 0) < $1
		   and count(distinct case when e.status = 'active' then e.id end) > 0
		order by avg_score asc
		limit 20
	`, threshold)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var results []strugglingLearnerRow
	for rows.Next() {
		var r strugglingLearnerRow
		if err := rows.Scan(&r.UserID, &r.UserName, &r.UserEmail, &r.Department, &r.AvgScore, &r.ActiveCourses, &r.LastActive); err != nil {
			continue
		}
		results = append(results, r)
	}
	if results == nil {
		results = []strugglingLearnerRow{}
	}
	c.JSON(http.StatusOK, results)
}

// completionRateRow represents per-department course completion rates.
type completionRateRow struct {
	DepartmentName string  `json:"department_name"`
	Enrolled       int     `json:"enrolled"`
	Completed      int     `json:"completed"`
	CompletionRate float64 `json:"completion_rate"`
}

// GetCompletionRates returns per-department course completion rates.
func (h *Handler) GetCompletionRates(c *gin.Context) {
	rows, err := h.Pool.Query(c.Request.Context(), `
		select
			coalesce(d.name, 'Unassigned') as department_name,
			count(distinct e.id)::int as enrolled,
			count(distinct case when e.status = 'completed' then e.id end)::int as completed,
			case when count(distinct e.id) > 0
				then round(count(distinct case when e.status = 'completed' then e.id end)::numeric / count(distinct e.id)::numeric * 100, 1)
				else 0
			end as completion_rate
		from users u
		left join user_departments ud on ud.user_id = u.id
		left join departments d on d.id = ud.department_id
		left join enrollments e on e.user_id = u.id
		where e.id is not null
		group by d.name
		order by completion_rate desc
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var results []completionRateRow
	for rows.Next() {
		var r completionRateRow
		if err := rows.Scan(&r.DepartmentName, &r.Enrolled, &r.Completed, &r.CompletionRate); err != nil {
			continue
		}
		results = append(results, r)
	}
	if results == nil {
		results = []completionRateRow{}
	}
	c.JSON(http.StatusOK, results)
}

// ----------------------------------------------------------------------
// #1 — Enrollment & completion dashboard (platform-wide)
// ----------------------------------------------------------------------

// enrollmentSummary covers the headline counts + funnel + averages.
type enrollmentSummary struct {
	TotalEnrollments    int     `json:"total_enrollments"`
	ActiveCount         int     `json:"active_count"`
	CompletedCount      int     `json:"completed_count"`
	DroppedCount        int     `json:"dropped_count"`
	PendingCount        int     `json:"pending_count"`
	CompletionRate      float64 `json:"completion_rate"`
	AvgProgress         float64 `json:"avg_progress"`
	UniqueLearners      int     `json:"unique_learners"`
	AvgDaysToComplete   float64 `json:"avg_days_to_complete"`
	TotalCoursesOffered int     `json:"total_courses_offered"`
}

// topCourseRow is a course ranked by enrollment volume with completion stats.
type topCourseRow struct {
	CourseID          int64   `json:"course_id"`
	CourseTitle       string  `json:"course_title"`
	Status            string  `json:"status"`
	Enrolled          int     `json:"enrolled"`
	Completed         int     `json:"completed"`
	CompletionRate    float64 `json:"completion_rate"`
	AvgProgress       float64 `json:"avg_progress"`
	AvgDaysToComplete float64 `json:"avg_days_to_complete"`
}

// GetEnrollmentOverview returns the platform-wide enrollment summary plus the
// top courses by enrollment. Used by the Platform tab of /analytics.
func (h *Handler) GetEnrollmentOverview(c *gin.Context) {
	var s enrollmentSummary

	// Headline counts + funnel in a single query.
	err := h.Pool.QueryRow(c.Request.Context(), `
		select
			count(*)::int,
			count(*) filter (where status = 'active')::int,
			count(*) filter (where status = 'completed')::int,
			count(*) filter (where status = 'dropped')::int,
			count(*) filter (where status = 'pending')::int,
			case when count(*) > 0
				then round(count(*) filter (where status = 'completed')::numeric / count(*)::numeric * 100, 1)
				else 0 end,
			coalesce(round(avg(progress_pct)::numeric, 1), 0),
			count(distinct user_id)::int,
			coalesce(round(extract(epoch from avg(completed_at - enrolled_at)) / 86400::numeric, 1), 0)
		from enrollments
	`).Scan(
		&s.TotalEnrollments, &s.ActiveCount, &s.CompletedCount,
		&s.DroppedCount, &s.PendingCount, &s.CompletionRate,
		&s.AvgProgress, &s.UniqueLearners, &s.AvgDaysToComplete,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Count of offered courses (anything that can be enrolled into).
	_ = h.Pool.QueryRow(c.Request.Context(),
		`select count(*)::int from courses`).Scan(&s.TotalCoursesOffered)

	// Top courses by enrollment volume.
	rows, err := h.Pool.Query(c.Request.Context(), `
		select
			c.id,
			c.title,
			c.status,
			count(e.id)::int as enrolled,
			count(e.id) filter (where e.status = 'completed')::int as completed,
			case when count(e.id) > 0
				then round(count(e.id) filter (where e.status = 'completed')::numeric / count(e.id)::numeric * 100, 1)
				else 0 end as completion_rate,
			coalesce(round(avg(e.progress_pct)::numeric, 1), 0) as avg_progress,
			coalesce(round(extract(epoch from avg(e.completed_at - e.enrolled_at)) / 86400::numeric, 1), 0) as avg_days
		from courses c
		left join enrollments e on e.course_id = c.id
		group by c.id, c.title, c.status
		order by enrolled desc
		limit 10
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	top := []topCourseRow{}
	for rows.Next() {
		var r topCourseRow
		if err := rows.Scan(&r.CourseID, &r.CourseTitle, &r.Status, &r.Enrolled, &r.Completed, &r.CompletionRate, &r.AvgProgress, &r.AvgDaysToComplete); err != nil {
			continue
		}
		top = append(top, r)
	}

	c.JSON(http.StatusOK, gin.H{
		"summary":     s,
		"top_courses": top,
	})
}

// enrollmentTimelinePoint is one bucket on the enrollments-over-time chart.
type enrollmentTimelinePoint struct {
	Date      string `json:"date"`
	Enrolled  int    `json:"enrolled"`
	Completed int    `json:"completed"`
}

// GetEnrollmentTimeline returns new enrollments + completions grouped by day
// for the last `days` days (default 30, max 365).
func (h *Handler) GetEnrollmentTimeline(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365
	}

	rows, err := h.Pool.Query(c.Request.Context(), `
		with bucket as (
			select d::date as day
			from generate_series(
				now()::date - ($1::int - 1) * interval '1 day',
				now()::date,
				interval '1 day'
			) d
		)
		select
			to_char(b.day, 'YYYY-MM-DD') as date,
			coalesce(count(e.id) filter (where e.enrolled_at::date = b.day), 0)::int as enrolled,
			coalesce(count(e.id) filter (where e.completed_at is not null and e.completed_at::date = b.day), 0)::int as completed
		from bucket b
		left join enrollments e on e.enrolled_at::date <= b.day
		group by b.day
		order by b.day asc
	`, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	out := []enrollmentTimelinePoint{}
	for rows.Next() {
		var p enrollmentTimelinePoint
		if err := rows.Scan(&p.Date, &p.Enrolled, &p.Completed); err != nil {
			continue
		}
		out = append(out, p)
	}
	c.JSON(http.StatusOK, out)
}

// stalledEnrollmentRow is an active enrollment with no recent lesson activity.
type stalledEnrollmentRow struct {
	UserID      int64   `json:"user_id"`
	UserName    string  `json:"user_name"`
	UserEmail   string  `json:"user_email"`
	CourseID    int64   `json:"course_id"`
	CourseTitle string  `json:"course_title"`
	Progress    float64 `json:"progress_pct"`
	EnrolledAt  string  `json:"enrolled_at"`
	LastActive  string  `json:"last_active"`
}

// GetStalledEnrollments lists active enrollments that have had no lesson
// activity in the last 14 days. Useful for spotting learners at risk of
// dropping out.
func (h *Handler) GetStalledEnrollments(c *gin.Context) {
	rows, err := h.Pool.Query(c.Request.Context(), `
		select
			u.id, u.name, u.email,
			co.id, co.title,
			coalesce(e.progress_pct, 0),
			to_char(e.enrolled_at, 'YYYY-MM-DD'),
			coalesce(to_char(max(lp.started_at), 'YYYY-MM-DD'), 'never')
		from enrollments e
		join users u on u.id = e.user_id
		join courses co on co.id = e.course_id
		left join lesson_progress lp on lp.user_id = e.user_id and lp.course_id = e.course_id
		where e.status = 'active'
		group by u.id, u.name, u.email, co.id, co.title, e.progress_pct, e.enrolled_at
		having max(lp.started_at) is null or max(lp.started_at) < now() - interval '14 days'
		order by e.enrolled_at asc
		limit 20
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	out := []stalledEnrollmentRow{}
	for rows.Next() {
		var r stalledEnrollmentRow
		if err := rows.Scan(&r.UserID, &r.UserName, &r.UserEmail, &r.CourseID, &r.CourseTitle, &r.Progress, &r.EnrolledAt, &r.LastActive); err != nil {
			continue
		}
		out = append(out, r)
	}
	c.JSON(http.StatusOK, out)
}

// ----------------------------------------------------------------------
// #2 — Certificates & badges reporting (platform-wide)
// ----------------------------------------------------------------------

// credentialSummary covers the headline counts and tier breakdowns for both
// certificates and badges.
type credentialSummary struct {
	CertificatesIssued int     `json:"certificates_issued"`
	BadgesAwarded      int     `json:"badges_awarded"`
	DistinctRecipients int     `json:"distinct_recipients"`
	AvgCertScore       float64 `json:"avg_cert_score"`
	CertTiers          struct {
		Bronze int `json:"bronze"`
		Silver int `json:"silver"`
		Gold   int `json:"gold"`
	} `json:"cert_tiers"`
	BadgeTiers struct {
		Bronze int `json:"bronze"`
		Silver int `json:"silver"`
		Gold   int `json:"gold"`
	} `json:"badge_tiers"`
}

// badgeCategoryRow is the count of awards grouped by badge category.
type badgeCategoryRow struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// topBadgeRow lists the most-awarded individual badges.
type topBadgeRow struct {
	BadgeID   int64  `json:"badge_id"`
	BadgeCode string `json:"badge_code"`
	BadgeName string `json:"badge_name"`
	Icon      string `json:"icon"`
	Tier      string `json:"tier"`
	Category  string `json:"category"`
	Awards    int    `json:"awards"`
}

// topEarnerRow lists the users with the most credentials (certs + badges).
type topEarnerRow struct {
	UserID           int64  `json:"user_id"`
	UserName         string `json:"user_name"`
	UserEmail        string `json:"user_email"`
	Certificates     int    `json:"certificates"`
	Badges           int    `json:"badges"`
	TotalCredentials int    `json:"total_credentials"`
}

// GetCredentialOverview returns the platform-wide certificates & badges summary
// plus top earners and the most-awarded badges. Used by the Platform tab.
func (h *Handler) GetCredentialOverview(c *gin.Context) {
	var s credentialSummary

	// Certificate counts + avg score in one query.
	var bronze, silver, gold int
	err := h.Pool.QueryRow(c.Request.Context(), `
				select
					count(*)::int,
					coalesce(round(avg(score_pct)::numeric, 1), 0),
					count(*) filter (where tier = 'bronze')::int,
					count(*) filter (where tier = 'silver')::int,
					count(*) filter (where tier = 'gold')::int
				from certificates
			`).Scan(&s.CertificatesIssued, &s.AvgCertScore, &bronze, &silver, &gold)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.CertTiers.Bronze = bronze
	s.CertTiers.Silver = silver
	s.CertTiers.Gold = gold

	// Badge counts + tier breakdown.
	var bBronze, bSilver, bGold int
	err = h.Pool.QueryRow(c.Request.Context(), `
				select
					count(*)::int,
					count(*) filter (where bd.tier = 'bronze')::int,
					count(*) filter (where bd.tier = 'silver')::int,
					count(*) filter (where bd.tier = 'gold')::int
				from badge_awards ba
				join badge_definitions bd on bd.id = ba.badge_id
			`).Scan(&s.BadgesAwarded, &bBronze, &bSilver, &bGold)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.BadgeTiers.Bronze = bBronze
	s.BadgeTiers.Silver = bSilver
	s.BadgeTiers.Gold = bGold

	// Distinct users who have at least one cert or badge.
	_ = h.Pool.QueryRow(c.Request.Context(), `
		select count(*)::int from (
			select user_id from certificates
			union
			select user_id from badge_awards
		) x
	`).Scan(&s.DistinctRecipients)

	// Top earners (certs + badges combined).
	topRows, err := h.Pool.Query(c.Request.Context(), `
		with cert_counts as (
			select user_id, count(*)::int as certs from certificates group by user_id
		), badge_counts as (
			select user_id, count(*)::int as badges from badge_awards group by user_id
		)
		select
			u.id,
			u.name,
			u.email,
			coalesce(cc.certs, 0)::int,
			coalesce(bc.badges, 0)::int,
			(coalesce(cc.certs, 0) + coalesce(bc.badges, 0))::int as total
		from users u
		join cert_counts cc on cc.user_id = u.id
		left join badge_counts bc on bc.user_id = u.id
		union
		select
			u.id,
			u.name,
			u.email,
			coalesce(cc.certs, 0)::int,
			coalesce(bc.badges, 0)::int,
			(coalesce(cc.certs, 0) + coalesce(bc.badges, 0))::int as total
		from users u
		join badge_counts bc on bc.user_id = u.id
		left join cert_counts cc on cc.user_id = u.id
		order by total desc
		limit 10
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer topRows.Close()

	topEarners := []topEarnerRow{}
	for topRows.Next() {
		var r topEarnerRow
		if err := topRows.Scan(&r.UserID, &r.UserName, &r.UserEmail, &r.Certificates, &r.Badges, &r.TotalCredentials); err != nil {
			continue
		}
		topEarners = append(topEarners, r)
	}

	// Most-awarded badges.
	badgeRows, err := h.Pool.Query(c.Request.Context(), `
				select
					bd.id,
					bd.code,
					bd.name,
					bd.icon,
					bd.tier,
					bd.category,
					count(ba.id)::int as awards
				from badge_definitions bd
				left join badge_awards ba on ba.badge_id = bd.id
				group by bd.id, bd.code, bd.name, bd.icon, bd.tier, bd.category
				order by awards desc
				limit 10
			`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer badgeRows.Close()

	topBadges := []topBadgeRow{}
	for badgeRows.Next() {
		var r topBadgeRow
		if err := badgeRows.Scan(&r.BadgeID, &r.BadgeCode, &r.BadgeName, &r.Icon, &r.Tier, &r.Category, &r.Awards); err != nil {
			continue
		}
		topBadges = append(topBadges, r)
	}

	// Badge awards by category.
	catRows, err := h.Pool.Query(c.Request.Context(), `
				select bd.category, count(ba.id)::int
				from badge_awards ba
				join badge_definitions bd on bd.id = ba.badge_id
				group by bd.category
				order by count(ba.id) desc
			`)
	var categories []badgeCategoryRow
	if err == nil {
		defer catRows.Close()
		for catRows.Next() {
			var r badgeCategoryRow
			if err := catRows.Scan(&r.Category, &r.Count); err != nil {
				continue
			}
			categories = append(categories, r)
		}
	}
	if categories == nil {
		categories = []badgeCategoryRow{}
	}

	c.JSON(http.StatusOK, gin.H{
		"summary":          s,
		"top_earners":      topEarners,
		"top_badges":       topBadges,
		"badge_categories": categories,
	})
}

// credentialTimelinePoint is one bucket on the credentials-over-time chart.
type credentialTimelinePoint struct {
	Date         string `json:"date"`
	Certificates int    `json:"certificates"`
	Badges       int    `json:"badges"`
}

// GetCredentialTimeline returns certificates + badges issued grouped by day
// for the last `days` days (default 30, max 365).
func (h *Handler) GetCredentialTimeline(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365
	}

	rows, err := h.Pool.Query(c.Request.Context(), `
				with bucket as (
					select d::date as day
					from generate_series(
						now()::date - ($1::int - 1) * interval '1 day',
						now()::date,
						interval '1 day'
					) d
				),
				certs as (
					select issued_at::date as day, count(*)::int as n
					from certificates
					where issued_at >= now()::date - ($1::int - 1) * interval '1 day'
					group by issued_at::date
				),
				badges as (
					select earned_at::date as day, count(*)::int as n
					from badge_awards
					where earned_at >= now()::date - ($1::int - 1) * interval '1 day'
					group by earned_at::date
				)
				select
					to_char(bk.day, 'YYYY-MM-DD') as date,
					coalesce(c.n, 0)::int as certificates,
					coalesce(bg.n, 0)::int as badges
				from bucket bk
				left join certs c on c.day = bk.day
				left join badges bg on bg.day = bk.day
				order by bk.day asc
			`, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	out := []credentialTimelinePoint{}
	for rows.Next() {
		var p credentialTimelinePoint
		if err := rows.Scan(&p.Date, &p.Certificates, &p.Badges); err != nil {
			continue
		}
		out = append(out, p)
	}
	c.JSON(http.StatusOK, out)
}

// ----------------------------------------------------------------------
// #3 — User growth & composition
// ----------------------------------------------------------------------

type roleCountRow struct {
	Role  string `json:"role"`
	Count int    `json:"count"`
}

type deptCountRow struct {
	Department string `json:"department"`
	Count      int    `json:"count"`
}

type userOverview struct {
	TotalUsers   int            `json:"total_users"`
	OrphanCount  int            `json:"orphan_count"`
	BulkCreated  int            `json:"bulk_created"`
	ByRole       []roleCountRow `json:"by_role"`
	ByDepartment []deptCountRow `json:"by_department"`
}

// GetUserOverview returns platform-wide user composition: total, orphans,
// bulk-created vs individual, by role, and by department.
func (h *Handler) GetUserOverview(c *gin.Context) {
	var o userOverview
	_ = h.Pool.QueryRow(c.Request.Context(),
		`select count(*)::int from users`).Scan(&o.TotalUsers)

	// Orphans: users not in any department.
	_ = h.Pool.QueryRow(c.Request.Context(), `
			select count(*)::int from users u
			where not exists (select 1 from user_departments ud where ud.user_id = u.id)
		`).Scan(&o.OrphanCount)

	// Bulk-created count from audit_log. Each users_bulk_created event records
	// created_count in details; sum across events.
	_ = h.Pool.QueryRow(c.Request.Context(), `
			select coalesce(sum((details->>'created_count')::int), 0)::int
			from audit_log
			where action = 'users_bulk_created'
		`).Scan(&o.BulkCreated)

	// By role (union of legacy users.role and user_roles).
	roleRows, err := h.Pool.Query(c.Request.Context(), `
			select role, count(*)::int from (
				select role from users
				union all
				select role from user_roles
			) x
			group by role
			order by count(*) desc
		`)
	o.ByRole = []roleCountRow{}
	if err == nil {
		defer roleRows.Close()
		for roleRows.Next() {
			var r roleCountRow
			if err := roleRows.Scan(&r.Role, &r.Count); err == nil {
				o.ByRole = append(o.ByRole, r)
			}
		}
	}

	// By department (including Unassigned).
	deptRows, err := h.Pool.Query(c.Request.Context(), `
			select
				coalesce(d.name, 'Unassigned') as department,
				count(distinct u.id)::int as count
			from users u
			left join user_departments ud on ud.user_id = u.id
			left join departments d on d.id = ud.department_id
			group by coalesce(d.name, 'Unassigned')
			order by count desc
		`)
	o.ByDepartment = []deptCountRow{}
	if err == nil {
		defer deptRows.Close()
		for deptRows.Next() {
			var r deptCountRow
			if err := deptRows.Scan(&r.Department, &r.Count); err == nil {
				o.ByDepartment = append(o.ByDepartment, r)
			}
		}
	}

	c.JSON(http.StatusOK, o)
}

type growthPoint struct {
	Date     string `json:"date"`
	NewUsers int    `json:"new_users"`
}

// GetUserGrowth returns new signups per day for the last `days` days.
func (h *Handler) GetUserGrowth(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365
	}

	rows, err := h.Pool.Query(c.Request.Context(), `
			with bucket as (
				select d::date as day
				from generate_series(
					now()::date - ($1::int - 1) * interval '1 day',
					now()::date,
					interval '1 day'
				) d
			)
			select
				to_char(bk.day, 'YYYY-MM-DD') as date,
				coalesce(count(u.id), 0)::int as new_users
			from bucket bk
			left join users u on u.created_at::date = bk.day
			group by bk.day
			order by bk.day asc
		`, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	out := []growthPoint{}
	for rows.Next() {
		var p growthPoint
		if err := rows.Scan(&p.Date, &p.NewUsers); err == nil {
			out = append(out, p)
		}
	}
	c.JSON(http.StatusOK, out)
}

// ----------------------------------------------------------------------
// #4 — Learning paths reporting
// ----------------------------------------------------------------------

type pathOverview struct {
	TotalPaths       int     `json:"total_paths"`
	PublishedPaths   int     `json:"published_paths"`
	TotalEnrollments int     `json:"total_enrollments"`
	CompletedCount   int     `json:"completed_count"`
	ActiveCount      int     `json:"active_count"`
	CompletionRate   float64 `json:"completion_rate"`
	AvgProgress      float64 `json:"avg_progress"`
}

type topPathRow struct {
	PathID         int64   `json:"path_id"`
	PathTitle      string  `json:"path_title"`
	Status         string  `json:"status"`
	Enrolled       int     `json:"enrolled"`
	Completed      int     `json:"completed"`
	CompletionRate float64 `json:"completion_rate"`
	AvgProgress    float64 `json:"avg_progress"`
}

// GetLearningPathOverview returns platform-wide learning path summary plus
// the top paths by enrollment.
func (h *Handler) GetLearningPathOverview(c *gin.Context) {
	var o pathOverview
	err := h.Pool.QueryRow(c.Request.Context(), `
			select
				count(distinct lp.id)::int,
				count(distinct lp.id) filter (where lp.status = 'published')::int,
				count(lpe.id)::int,
				count(lpe.id) filter (where lpe.status = 'completed')::int,
				count(lpe.id) filter (where lpe.status = 'active')::int,
				case when count(lpe.id) > 0
					then round(count(lpe.id) filter (where lpe.status = 'completed')::numeric / count(lpe.id)::numeric * 100, 1)
					else 0 end,
				coalesce(round(avg(lpe.progress_pct)::numeric, 1), 0)
			from learning_paths lp
			left join learning_path_enrollments lpe on lpe.learning_path_id = lp.id
		`).Scan(
		&o.TotalPaths, &o.PublishedPaths, &o.TotalEnrollments,
		&o.CompletedCount, &o.ActiveCount, &o.CompletionRate, &o.AvgProgress,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rows, err := h.Pool.Query(c.Request.Context(), `
			select
				lp.id,
				lp.title,
				lp.status,
				count(lpe.id)::int as enrolled,
				count(lpe.id) filter (where lpe.status = 'completed')::int as completed,
				case when count(lpe.id) > 0
					then round(count(lpe.id) filter (where lpe.status = 'completed')::numeric / count(lpe.id)::numeric * 100, 1)
					else 0 end as completion_rate,
				coalesce(round(avg(lpe.progress_pct)::numeric, 1), 0) as avg_progress
			from learning_paths lp
			left join learning_path_enrollments lpe on lpe.learning_path_id = lp.id
			group by lp.id, lp.title, lp.status
			order by enrolled desc
			limit 10
		`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	top := []topPathRow{}
	for rows.Next() {
		var r topPathRow
		if err := rows.Scan(&r.PathID, &r.PathTitle, &r.Status, &r.Enrolled, &r.Completed, &r.CompletionRate, &r.AvgProgress); err == nil {
			top = append(top, r)
		}
	}

	c.JSON(http.StatusOK, gin.H{"summary": o, "top_paths": top})
}

// ----------------------------------------------------------------------
// #5 — Content reporting
// ----------------------------------------------------------------------

type contentOverview struct {
	CoursesTotal        int `json:"courses_total"`
	CoursesPublished    int `json:"courses_published"`
	CoursesDraft        int `json:"courses_draft"`
	CoursesArchived     int `json:"courses_archived"`
	DocumentsTotal      int `json:"documents_total"`
	DocumentsReady      int `json:"documents_ready"`
	DocumentsProcessing int `json:"documents_processing"`
	DocumentsFailed     int `json:"documents_failed"`
}

type creatorCountRow struct {
	CreatorID   int64  `json:"creator_id"`
	CreatorName string `json:"creator_name"`
	Courses     int    `json:"courses"`
	Documents   int    `json:"documents"`
}

// GetContentOverview returns content inventory counts + review queue health.
func (h *Handler) GetContentOverview(c *gin.Context) {
	var o contentOverview
	err := h.Pool.QueryRow(c.Request.Context(), `
			select
				count(*)::int,
				count(*) filter (where status = 'published')::int,
				count(*) filter (where status = 'draft')::int,
				count(*) filter (where status = 'archived')::int
			from courses
		`).Scan(&o.CoursesTotal, &o.CoursesPublished, &o.CoursesDraft, &o.CoursesArchived)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_ = h.Pool.QueryRow(c.Request.Context(), `
			select
				count(*)::int,
				count(*) filter (where status = 'ready')::int,
				count(*) filter (where status = 'processing')::int,
				count(*) filter (where status = 'failed')::int
			from documents
		`).Scan(&o.DocumentsTotal, &o.DocumentsReady, &o.DocumentsProcessing, &o.DocumentsFailed)

	// Review queue health from audit / courses. Use courses review_status if present.
	review := map[string]int{}
	revRows, err := h.Pool.Query(c.Request.Context(), `
			select coalesce(review_status, 'none'), count(*)::int
			from courses
			group by coalesce(review_status, 'none')
		`)
	if err == nil {
		defer revRows.Close()
		for revRows.Next() {
			var status string
			var count int
			if err := revRows.Scan(&status, &count); err == nil {
				review[status] = count
			}
		}
	}

	// Content authored per creator (courses + documents combined).
	creatorRows, err := h.Pool.Query(c.Request.Context(), `
			with course_creators as (
				select created_by as user_id, count(*)::int as n from courses group by created_by
			), doc_uploaders as (
				select uploaded_by as user_id, count(*)::int as n from documents where uploaded_by is not null group by uploaded_by
			)
			select
				u.id,
				coalesce(u.name, 'Unknown'),
				coalesce(cc.n, 0)::int,
				coalesce(du.n, 0)::int
			from users u
			left join course_creators cc on cc.user_id = u.id
			left join doc_uploaders du on du.user_id = u.id
			where cc.n is not null or du.n is not null
			order by (coalesce(cc.n, 0) + coalesce(du.n, 0)) desc
			limit 10
		`)
	creators := []creatorCountRow{}
	if err == nil {
		defer creatorRows.Close()
		for creatorRows.Next() {
			var r creatorCountRow
			if err := creatorRows.Scan(&r.CreatorID, &r.CreatorName, &r.Courses, &r.Documents); err == nil {
				creators = append(creators, r)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"summary":       o,
		"review_health": review,
		"top_creators":  creators,
	})
}

// ----------------------------------------------------------------------
// #6 — Engagement refinement
// ----------------------------------------------------------------------

type engagementOverview struct {
	ActiveStreaks         int     `json:"active_streaks"`
	AvgStreakLength       float64 `json:"avg_streak_length"`
	LongestStreak         int     `json:"longest_streak"`
	TotalPracticeSessions int     `json:"total_practice_sessions"`
	AvgPracticeScore      float64 `json:"avg_practice_score"`
	UnreadNotifications   int     `json:"unread_notifications"`
}

// GetEngagementOverview returns gamification + engagement stats.
func (h *Handler) GetEngagementOverview(c *gin.Context) {
	var o engagementOverview

	// Streaks: distinct users with a streak entry in the last 7 days count as active.
	_ = h.Pool.QueryRow(c.Request.Context(), `
			select
				count(distinct user_id)::int,
				coalesce(round(avg(per_user_len)::numeric, 1), 0),
				coalesce(max(per_user_len), 0)::int
			from (
				select user_id, count(*)::int as per_user_len
				from user_streaks
				where streak_date >= now()::date - interval '7 days'
				group by user_id
			) s
		`).Scan(&o.ActiveStreaks, &o.AvgStreakLength, &o.LongestStreak)

	_ = h.Pool.QueryRow(c.Request.Context(), `
			select
				count(*)::int,
				coalesce(round(avg(score_pct)::numeric, 1), 0)
			from practice_sessions
			where completed_at is not null
		`).Scan(&o.TotalPracticeSessions, &o.AvgPracticeScore)

	_ = h.Pool.QueryRow(c.Request.Context(),
		`select count(*)::int from notifications where is_read = false`).Scan(&o.UnreadNotifications)

	c.JSON(http.StatusOK, o)
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/analytics/overview", h.GetOverview)
	r.GET("/analytics/most-failed", h.GetMostFailedTopics)
	r.GET("/analytics/course-effectiveness", h.GetCourseEffectiveness)
	r.GET("/analytics/coach-usage", h.GetCoachUsageOverTime)
	r.GET("/analytics/audit-log", h.GetAuditLogs)
	r.GET("/analytics/enrollments/overview", h.GetEnrollmentOverview)
	r.GET("/analytics/enrollments/timeline", h.GetEnrollmentTimeline)
	r.GET("/analytics/enrollments/stalled", h.GetStalledEnrollments)
	r.GET("/analytics/credentials/overview", h.GetCredentialOverview)
	r.GET("/analytics/credentials/timeline", h.GetCredentialTimeline)
	r.GET("/analytics/users/overview", h.GetUserOverview)
	r.GET("/analytics/users/growth", h.GetUserGrowth)
	r.GET("/analytics/learning-paths/overview", h.GetLearningPathOverview)
	r.GET("/analytics/content/overview", h.GetContentOverview)
	r.GET("/analytics/engagement/overview", h.GetEngagementOverview)
}

// RegisterManagerRoutes adds team-focused analytics endpoints for managers.
func (h *Handler) RegisterManagerRoutes(r *gin.RouterGroup) {
	r.GET("/analytics/team-overview", h.GetTeamOverview)
	r.GET("/analytics/struggling-learners", h.GetStrugglingLearners)
	r.GET("/analytics/completion-rates", h.GetCompletionRates)
}
