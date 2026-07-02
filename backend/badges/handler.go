package badges

import (
	"context"
	"encoding/json"
	"fgb-lp/audit"
	database "fgb-lp/database/queries"
	"net/http"

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

// BadgeDefinition represents a badge template from the database.
type BadgeDefinition struct {
	ID          int64           `json:"id"`
	Code        string          `json:"code"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Icon        string          `json:"icon"`
	Tier        string          `json:"tier"`
	Category    string          `json:"category"`
	Criteria    json.RawMessage `json:"criteria"`
}

// EarnedBadge represents a badge earned by a user.
type EarnedBadge struct {
	ID          int64  `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Tier        string `json:"tier"`
	Category    string `json:"category"`
	EarnedAt    string `json:"earned_at"`
}

// BadgeWithEarned includes earned status for the current user.
type BadgeWithEarned struct {
	ID          int64   `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Icon        string  `json:"icon"`
	Tier        string  `json:"tier"`
	Category    string  `json:"category"`
	Earned      bool    `json:"earned"`
	EarnedAt    *string `json:"earned_at,omitempty"`
}

// ListDefinitions returns all badge definitions with earned status for the current user.
// GET /badges
func (h *Handler) ListDefinitions(c *gin.Context) {
	userID := c.GetInt64("user_id")

	rows, err := h.Pool.Query(c.Request.Context(), `
		SELECT bd.id, bd.code, bd.name, bd.description, bd.icon, bd.tier, bd.category,
		       ba.earned_at IS NOT NULL AS earned,
		       ba.earned_at
		FROM badge_definitions bd
		LEFT JOIN badge_awards ba ON ba.badge_id = bd.id AND ba.user_id = $1
		ORDER BY bd.id`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch badges"})
		return
	}
	defer rows.Close()

	out := []BadgeWithEarned{}
	for rows.Next() {
		var b BadgeWithEarned
		var earnedAt *string
		if err := rows.Scan(&b.ID, &b.Code, &b.Name, &b.Description, &b.Icon, &b.Tier, &b.Category,
			&b.Earned, &earnedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan badge"})
			return
		}
		if earnedAt != nil {
			t := string(*earnedAt)
			b.EarnedAt = &t
		}
		out = append(out, b)
	}
	if out == nil {
		out = []BadgeWithEarned{}
	}

	c.JSON(http.StatusOK, out)
}

// ListEarned returns badges earned by the current user with badge definition details.
// GET /badges/earned
func (h *Handler) ListEarned(c *gin.Context) {
	userID := c.GetInt64("user_id")

	rows, err := h.Pool.Query(c.Request.Context(), `
		SELECT bd.id, bd.code, bd.name, bd.description, bd.icon, bd.tier, bd.category,
		       ba.earned_at::text
		FROM badge_awards ba
		JOIN badge_definitions bd ON bd.id = ba.badge_id
		WHERE ba.user_id = $1
		ORDER BY ba.earned_at DESC`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch earned badges"})
		return
	}
	defer rows.Close()

	out := []EarnedBadge{}
	for rows.Next() {
		var b EarnedBadge
		var earnedAt string
		if err := rows.Scan(&b.ID, &b.Code, &b.Name, &b.Description, &b.Icon, &b.Tier, &b.Category, &earnedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan earned badge"})
			return
		}
		b.EarnedAt = earnedAt
		out = append(out, b)
	}
	if out == nil {
		out = []EarnedBadge{}
	}

	c.JSON(http.StatusOK, out)
}

// EarnedCount returns the count of badges earned by the current user.
// GET /badges/earned-count
func (h *Handler) EarnedCount(c *gin.Context) {
	userID := c.GetInt64("user_id")

	var count int64
	err := h.Pool.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM badge_awards WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count badges"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// BadgeCriteria represents the parsed criteria JSON from a badge definition.
type BadgeCriteria struct {
	Type      string  `json:"type"`
	Count     int     `json:"count,omitempty"`
	Threshold float64 `json:"threshold,omitempty"`
	MaxDays   int     `json:"max_days,omitempty"`
}

// EvaluateBadges checks all badge criteria against a user's current state
// and awards any newly-earned badges. Call this after a user completes a course,
// scores points, or hits a streak milestone.
func (h *Handler) EvaluateBadges(ctx context.Context, userID int64) ([]EarnedBadge, error) {
	rows, err := h.Pool.Query(ctx,
		`SELECT id, code, name, description, icon, tier, category, criteria FROM badge_definitions`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	defs := []struct {
		ID          int64
		Code        string
		Name        string
		Description string
		Icon        string
		Tier        string
		Category    string
		Criteria    json.RawMessage
	}{}
	for rows.Next() {
		var d struct {
			ID          int64
			Code        string
			Name        string
			Description string
			Icon        string
			Tier        string
			Category    string
			Criteria    json.RawMessage
		}
		if err := rows.Scan(&d.ID, &d.Code, &d.Name, &d.Description, &d.Icon, &d.Tier, &d.Category, &d.Criteria); err != nil {
			return nil, err
		}
		defs = append(defs, d)
	}

	awarded := []EarnedBadge{}

	for _, def := range defs {
		if len(def.Criteria) == 0 {
			continue
		}

		var crit BadgeCriteria
		if err := json.Unmarshal(def.Criteria, &crit); err != nil {
			continue
		}

		earned := false

		switch crit.Type {
		case "courses_completed":
			var count int64
			err = h.Pool.QueryRow(ctx,
				`SELECT COUNT(*) FROM enrollments WHERE user_id = $1 AND status = 'completed'`, userID).Scan(&count)
			if err == nil && count >= int64(crit.Count) {
				earned = true
			}

		case "streak":
			streak, err := h.Queries.GetUserStreak(ctx, userID)
			if err == nil {
				days := int64(0)
				if v, ok := streak.(int64); ok {
					days = v
				}
				if days >= int64(crit.Count) {
					earned = true
				}
			}

		case "perfect_score":
			var count int64
			err = h.Pool.QueryRow(ctx,
				`SELECT COUNT(*) FROM course_scores WHERE user_id = $1 AND score >= 100`, userID).Scan(&count)
			if err == nil && count > 0 {
				earned = true
			}

		case "score_above":
			var count int64
			err = h.Pool.QueryRow(ctx,
				`SELECT COUNT(*) FROM course_scores WHERE user_id = $1 AND score >= $2`, userID, crit.Threshold).Scan(&count)
			if err == nil && count > 0 {
				earned = true
			}

		case "speed_days":
			var count int64
			err = h.Pool.QueryRow(ctx, `
				SELECT COUNT(*) FROM enrollments
				WHERE user_id = $1
				  AND status = 'completed'
				  AND completed_at IS NOT NULL
				  AND enrolled_at IS NOT NULL
				  AND EXTRACT(EPOCH FROM (completed_at - enrolled_at)) / 86400 <= $2::float`,
				userID, crit.MaxDays).Scan(&count)
			if err == nil && count > 0 {
				earned = true
			}

		case "total_xp":
			total, err := h.Queries.GetUserTotalScore(ctx, userID)
			if err == nil {
				var totalVal int64
				if total != nil {
					if v, ok := total.(int64); ok {
						totalVal = v
					}
				}
				if totalVal >= int64(crit.Count) {
					earned = true
				}
			}
		}

		if earned {
			_, err = h.Pool.Exec(ctx, `
				INSERT INTO badge_awards (user_id, badge_id)
				VALUES ($1, $2)
				ON CONFLICT DO NOTHING`, userID, def.ID)
			if err != nil {
				continue
			}

			var earnedAt string
			err = h.Pool.QueryRow(ctx,
				`SELECT earned_at::text FROM badge_awards WHERE user_id = $1 AND badge_id = $2`,
				userID, def.ID).Scan(&earnedAt)
			if err == nil {
				awarded = append(awarded, EarnedBadge{
					ID:          def.ID,
					Code:        def.Code,
					Name:        def.Name,
					Description: def.Description,
					Icon:        def.Icon,
					Tier:        def.Tier,
					Category:    def.Category,
					EarnedAt:    earnedAt,
				})

				audit.Logf(h.Queries, nil, "badge_awarded",
					"user %d earned badge %q (type=%s)", userID, def.Name, crit.Type)
			}
		}
	}

	if awarded == nil {
		awarded = []EarnedBadge{}
	}

	return awarded, nil
}

// RegisterRoutes registers protected badge routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/badges", h.ListDefinitions)
	r.GET("/badges/earned", h.ListEarned)
	r.GET("/badges/earned-count", h.EarnedCount)
}
