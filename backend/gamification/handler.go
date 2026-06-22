package gamification

import (
	database "fgb-lp/database/queries"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Queries *database.Queries
}

func NewHandler(queries *database.Queries) *Handler {
	return &Handler{Queries: queries}
}

// RecordEngagement logs today as an active streak day for the current user.
// Called whenever a user opens a course, starts a lesson, or practices.
func (h *Handler) RecordEngagement(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if err := h.Queries.RecordStreak(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// GetStreak returns the current streak count (consecutive days) for the user.
func (h *Handler) GetStreak(c *gin.Context) {
	userID := c.GetInt64("user_id")
	streak, err := h.Queries.GetUserStreak(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// sqlc returns interface{} for coalesce expressions; pgx gives int64
	days := int64(0)
	if v, ok := streak.(int64); ok {
		days = v
	}
	c.JSON(http.StatusOK, gin.H{"streak_days": days})
}

// ScoreCourse assigns a score to a user for a course and returns the updated row.
// The frontend calls this when a learner completes a course or scores points.
func (h *Handler) ScoreCourse(c *gin.Context) {
	var body struct {
		CourseID  int64 `json:"course_id" binding:"required"`
		Score     int32 `json:"score" binding:"required"`
		Completed bool  `json:"completed"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetInt64("user_id")

	row, err := h.Queries.UpsertCourseScore(c.Request.Context(),
		database.UpsertCourseScoreParams{
			UserID:   userID,
			CourseID: body.CourseID,
			Score:    body.Score,
			Column4:  body.Completed,
		})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, row)
}

// GetTotalScore returns the user's total score across all courses.
func (h *Handler) GetTotalScore(c *gin.Context) {
	userID := c.GetInt64("user_id")
	total, err := h.Queries.GetUserTotalScore(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"total_score": total})
}

// GetLeaderboard returns the top N users by total score.
func (h *Handler) GetLeaderboard(c *gin.Context) {
	var query struct {
		Limit int32 `form:"limit"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		query.Limit = 10
	}
	if query.Limit <= 0 || query.Limit > 50 {
		query.Limit = 10
	}

	userID := c.GetInt64("user_id")
	rows, err := h.Queries.GetLeaderboard(c.Request.Context(), query.Limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type entry struct {
		UserID     int64  `json:"user_id"`
		Name       string `json:"name"`
		TotalScore int32  `json:"total_score"`
		IsMe       bool   `json:"is_me"`
	}

	out := make([]entry, 0, len(rows))
	for _, r := range rows {
		out = append(out, entry{
			UserID:     r.UserID,
			Name:       r.UserName,
			TotalScore: r.TotalScore,
			IsMe:       r.UserID == userID,
		})
	}
	if out == nil {
		out = []entry{}
	}
	c.JSON(http.StatusOK, gin.H{"leaderboard": out})
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/gamification/streak", h.RecordEngagement)
	r.GET("/gamification/streak", h.GetStreak)
	r.POST("/gamification/score", h.ScoreCourse)
	r.GET("/gamification/score", h.GetTotalScore)
	r.GET("/gamification/leaderboard", h.GetLeaderboard)
}
