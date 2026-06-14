package analytics

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

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/analytics/overview", h.GetOverview)
	r.GET("/analytics/most-failed", h.GetMostFailedTopics)
	r.GET("/analytics/course-effectiveness", h.GetCourseEffectiveness)
	r.GET("/analytics/coach-usage", h.GetCoachUsageOverTime)
}
