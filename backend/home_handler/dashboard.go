package homehandler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DashboardHandler serves the personalised dashboard endpoint.
type DashboardHandler struct {
	Pool *pgxpool.Pool
}

// NewDashboardHandler creates a DashboardHandler backed by the given pool.
func NewDashboardHandler(pool *pgxpool.Pool) *DashboardHandler {
	return &DashboardHandler{Pool: pool}
}

// GetDashboard returns the aggregated data the frontend dashboard consumes.
func (h *DashboardHandler) GetDashboard(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	uid := userID.(int64)
	ctx := c.Request.Context()

	resp := gin.H{}

	// ---- continue_learning: most recent in-progress course ----
	type continueLearning struct {
		ID            int64  `json:"id"`
		Title         string `json:"title"`
		CurrentModule int    `json:"current_module"`
		ProgressPct   int    `json:"progress_pct"`
		Image         string `json:"image"`
	}
	{
		var cl continueLearning
		var totalMods int
		err := h.Pool.QueryRow(ctx, `
			SELECT c.id, c.title, lp.current_module,
			       (SELECT count(*) FROM modules WHERE course_id = c.id) AS total_modules
			FROM lesson_progress lp
			JOIN courses c ON c.id = lp.course_id
			WHERE lp.user_id = $1 AND lp.completed = false
			ORDER BY lp.started_at DESC
			LIMIT 1
		`, uid).Scan(&cl.ID, &cl.Title, &cl.CurrentModule, &totalMods)
		if err == nil {
			if totalMods > 0 {
				cl.ProgressPct = cl.CurrentModule * 100 / totalMods
			}
			cl.Image = pathImg(cl.Title)
			resp["continue_learning"] = cl
		}
	}

	// ---- recommended: published courses the user hasn't started ----
	type recommendedCourse struct {
		Title    string `json:"title"`
		Duration string `json:"duration"`
		Level    string `json:"level"`
		Image    string `json:"image"`
	}
	{
		recRows, err := h.Pool.Query(ctx, `
			SELECT c.title,
			       coalesce((SELECT count(*) FROM modules WHERE course_id = c.id), 0) AS module_count
			FROM courses c
			WHERE c.status = 'published' AND c.approved = true
			  AND c.id NOT IN (SELECT course_id FROM lesson_progress WHERE user_id = $1)
			ORDER BY c.updated_at DESC
			LIMIT 3
		`, uid)
		if err == nil {
			var recs []recommendedCourse
			for recRows.Next() {
				var rc recommendedCourse
				var mc int
				if err := recRows.Scan(&rc.Title, &mc); err != nil {
					continue
				}
				rc.Duration = durFromModules(mc)
				rc.Level = levelFromModules(mc)
				rc.Image = pathImg(rc.Title)
				recs = append(recs, rc)
			}
			recRows.Close()
			if len(recs) > 0 {
				resp["recommended"] = recs
			}
		}
	}

	// ---- achievements: derived from real user data ----
	type achievement struct {
		Label string `json:"label"`
		Tier  string `json:"tier"`
		Icon  string `json:"icon"`
	}
	{
		var achievements []achievement

		var completedCount int
		h.Pool.QueryRow(ctx,
			`SELECT count(*) FROM lesson_progress WHERE user_id = $1 AND completed = true`, uid,
		).Scan(&completedCount)
		if completedCount >= 1 {
			achievements = append(achievements, achievement{
				Label: "Quick Learner",
				Tier:  tierFor(completedCount, 1, 3, 5),
				Icon:  "zap",
			})
		}

		var streakDays int
		h.Pool.QueryRow(ctx, `
			WITH dates AS (
			  SELECT streak_date,
			    streak_date - (row_number() OVER (ORDER BY streak_date DESC))::int AS grp
			  FROM user_streaks
			  WHERE user_id = $1 AND streak_date <= current_date
			),
			consecutive AS (
			  SELECT count(*) AS cnt FROM dates
			  WHERE grp = (SELECT grp FROM dates WHERE streak_date = (SELECT max(streak_date) FROM dates))
			)
			SELECT coalesce((SELECT cnt FROM consecutive), 0)
		`, uid).Scan(&streakDays)
		if streakDays >= 3 {
			achievements = append(achievements, achievement{
				Label: fmt.Sprintf("%d-Day Streak", streakDays),
				Tier:  tierFor(streakDays, 3, 7, 14),
				Icon:  "flame",
			})
		}

		var totalScore int
		h.Pool.QueryRow(ctx,
			`SELECT coalesce(sum(score), 0) FROM course_scores WHERE user_id = $1`, uid,
		).Scan(&totalScore)
		if totalScore >= 50 {
			achievements = append(achievements, achievement{
				Label: "Knowledge Explorer",
				Tier:  tierFor(totalScore, 50, 200, 500),
				Icon:  "book-open",
			})
		}

		if completedCount >= 3 {
			achievements = append(achievements, achievement{
				Label: "Dedicated Learner",
				Tier:  tierFor(completedCount, 3, 7, 12),
				Icon:  "users",
			})
		}

		if len(achievements) > 0 {
			resp["achievements"] = achievements
		}
	}

	// ---- learning_path: recent courses with status ----
	type pathStep struct {
		Label  string `json:"label"`
		Status string `json:"status"`
	}
	{
		pathRows, err := h.Pool.Query(ctx, `
			SELECT c.title,
			       CASE WHEN lp.completed THEN 'Completed'
			            WHEN lp.current_module > 0 THEN 'In Progress'
			            ELSE 'Not Started'
			       END AS status
			FROM lesson_progress lp
			JOIN courses c ON c.id = lp.course_id
			WHERE lp.user_id = $1
			ORDER BY lp.completed ASC, lp.started_at DESC
			LIMIT 5
		`, uid)
		if err == nil {
			var path []pathStep
			for pathRows.Next() {
				var ps pathStep
				if err := pathRows.Scan(&ps.Label, &ps.Status); err != nil {
					continue
				}
				path = append(path, ps)
			}
			pathRows.Close()
			if len(path) > 0 {
				resp["learning_path"] = path
			}
		}
	}

	// ---- weekly_activity: last 7 days of item interactions ----
	{
		var weekly []int
		hasActivity := false
		for i := 6; i >= 0; i-- {
			var cnt int
			h.Pool.QueryRow(ctx, `
				SELECT count(*) FROM item_progress
				WHERE user_id = $1 AND answered_at::date = (current_date - $2::int)
			`, uid, i).Scan(&cnt)
			weekly = append(weekly, cnt)
			if cnt > 0 {
				hasActivity = true
			}
		}
		if hasActivity {
			resp["weekly_activity"] = weekly
		}
	}

	// ---- skill_radar: topic performance from practice sessions ----
	{
		radarRows, err := h.Pool.Query(ctx, `
			SELECT topic,
			       CASE WHEN sum(question_count) > 0
			            THEN round(sum(correct_count)::numeric / sum(question_count) * 100)
			            ELSE 0
			       END::int AS pct
			FROM practice_sessions
			WHERE user_id = $1 AND completed_at IS NOT NULL
			GROUP BY topic
			ORDER BY pct DESC
			LIMIT 5
		`, uid)
		if err == nil {
			var labels []string
			var data []int
			for radarRows.Next() {
				var label string
				var pct int
				if err := radarRows.Scan(&label, &pct); err != nil {
					continue
				}
				labels = append(labels, label)
				data = append(data, pct)
			}
			radarRows.Close()
			if len(labels) > 0 {
				resp["skill_radar"] = gin.H{"labels": labels, "data": data}
			}
		}
	}

	// ---- featured_learning: recently published course titles ----
	{
		featRows, err := h.Pool.Query(ctx, `
			SELECT title FROM courses
			WHERE status = 'published' AND approved = true
			ORDER BY updated_at DESC
			LIMIT 6
		`)
		if err == nil {
			var featured []string
			for featRows.Next() {
				var t string
				if err := featRows.Scan(&t); err != nil {
					continue
				}
				featured = append(featured, t)
			}
			featRows.Close()
			if len(featured) > 0 {
				resp["featured_learning"] = featured
			}
		}
	}

	// ---- deadlines: in-progress courses (oldest first) ----
	type deadlineItem struct {
		Title string `json:"title"`
		Due   string `json:"due"`
		Icon  string `json:"icon"`
	}
	{
		dlRows, err := h.Pool.Query(ctx, `
			SELECT c.title, lp.started_at
			FROM lesson_progress lp
			JOIN courses c ON c.id = lp.course_id
			WHERE lp.user_id = $1 AND lp.completed = false
			ORDER BY lp.started_at ASC
			LIMIT 2
		`, uid)
		if err == nil {
			var dls []deadlineItem
			for dlRows.Next() {
				var dl deadlineItem
				var startedAt time.Time
				if err := dlRows.Scan(&dl.Title, &startedAt); err != nil {
					continue
				}
				daysSince := int(time.Since(startedAt).Hours() / 24)
				if daysSince > 0 {
					dl.Due = fmt.Sprintf("Started %d days ago", daysSince)
					dl.Icon = "clock"
					dls = append(dls, dl)
				}
			}
			dlRows.Close()
			if len(dls) > 0 {
				resp["deadlines"] = dls
			}
		}
	}

	// ---- whats_new: recently published content ----
	type whatsNewItem struct {
		Title string `json:"title"`
		Meta  string `json:"meta"`
		Icon  string `json:"icon"`
	}
	{
		wnRows, err := h.Pool.Query(ctx, `
			(SELECT title, 'Just added' AS meta, 'sparkles' AS icon, updated_at
			 FROM courses WHERE status = 'published' AND approved = true ORDER BY updated_at DESC LIMIT 1)
			UNION ALL
			(SELECT title, 'Recommended for you' AS meta, 'trending-up' AS icon, updated_at
			 FROM courses WHERE status = 'published' AND approved = true
			   AND id NOT IN (SELECT course_id FROM lesson_progress WHERE user_id = $1)
			 ORDER BY updated_at DESC LIMIT 1)
		`, uid)
		if err == nil {
			var wns []whatsNewItem
			for wnRows.Next() {
				var wn whatsNewItem
				var ts time.Time
				if err := wnRows.Scan(&wn.Title, &wn.Meta, &wn.Icon, &ts); err != nil {
					continue
				}
				wns = append(wns, wn)
			}
			wnRows.Close()
			if len(wns) > 0 {
				resp["whats_new"] = wns
			}
		}
	}

	c.JSON(http.StatusOK, resp)
}

// ---- helpers ----

// pathImg resolves a course title to a pathway image slug.
// Mirrors the frontend HINTS table in +page.svelte.
func pathImg(title string) string {
	lower := strings.ToLower(title)
	switch {
	case strings.Contains(lower, "cyber") || strings.Contains(lower, "security") ||
		strings.Contains(lower, "fraud") || strings.Contains(lower, "infosec") ||
		strings.Contains(lower, "information"):
		return "/brand/pathways/cybersecurity.png"
	case strings.Contains(lower, "compliance") || strings.Contains(lower, "regulator") ||
		strings.Contains(lower, "policy") || strings.Contains(lower, "aml") ||
		strings.Contains(lower, "privacy"):
		return "/brand/pathways/compliance.png"
	case strings.Contains(lower, "risk") || strings.Contains(lower, "credit") ||
		strings.Contains(lower, "portfolio"):
		return "/brand/pathways/risk-credit.png"
	case strings.Contains(lower, "leader") || strings.Contains(lower, "executive") ||
		strings.Contains(lower, "management") || strings.Contains(lower, "team"):
		return "/brand/pathways/leadership.png"
	case strings.Contains(lower, "customer") || strings.Contains(lower, "service") ||
		strings.Contains(lower, "relationship") || strings.Contains(lower, "experience") ||
		strings.Contains(lower, "retail"):
		return "/brand/pathways/customer-service.png"
	default:
		return "/brand/pathways/banking-foundations.png"
	}
}

func durFromModules(moduleCount int) string {
	mins := moduleCount * 12
	if mins < 10 {
		mins = 10
	}
	return fmt.Sprintf("%d min", mins)
}

func levelFromModules(moduleCount int) string {
	switch {
	case moduleCount <= 3:
		return "Beginner"
	case moduleCount <= 6:
		return "Intermediate"
	default:
		return "Advanced"
	}
}

func tierFor(value, bronze, silver, gold int) string {
	if value >= gold {
		return "gold"
	}
	if value >= silver {
		return "silver"
	}
	return "bronze"
}
