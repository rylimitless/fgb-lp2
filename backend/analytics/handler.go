package analytics

import (
	"encoding/json"
	database "fgb-lp/database/queries"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Queries *database.Queries
}

func NewHandler(queries *database.Queries) *Handler {
	return &Handler{Queries: queries}
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
	data, err := h.Queries.GetCoachQueriesOverTime(c.Request.Context(), 30)
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

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/analytics/overview", h.GetOverview)
	r.GET("/analytics/most-failed", h.GetMostFailedTopics)
	r.GET("/analytics/course-effectiveness", h.GetCourseEffectiveness)
	r.GET("/analytics/coach-usage", h.GetCoachUsageOverTime)
	r.GET("/analytics/audit-log", h.GetAuditLogs)
}
