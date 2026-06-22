package lessons

import (
	"encoding/json"
	database "fgb-lp/database/queries"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	Queries *database.Queries
}

func NewHandler(queries *database.Queries) *Handler {
	return &Handler{Queries: queries}
}

func (h *Handler) ListPublished(c *gin.Context) {
	query := c.Query("q")
	userID := c.GetInt64("user_id")

	var courses []database.Course
	var err error

	if strings.TrimSpace(query) != "" {
		rows, searchErr := h.Queries.SearchPublishedCourses(c.Request.Context(),
			database.SearchPublishedCoursesParams{PlaintoTsquery: query, Limit: 50})
		if searchErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": searchErr.Error()})
			return
		}
		for _, r := range rows {
			courses = append(courses, database.Course{
				ID: r.ID, Title: r.Title, Description: r.Description,
				CreatedBy: r.CreatedBy, SourceDocIds: r.SourceDocIds,
				Status: r.Status, Settings: r.Settings,
				CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
				Department: r.Department, Approved: r.Approved,
				ReviewStatus: r.ReviewStatus, ReviewNotes: r.ReviewNotes,
			})
		}
	} else {
		courses, err = h.Queries.GetPublishedCourses(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if courses == nil {
		courses = []database.Course{}
	}

	progressList, _ := h.Queries.GetAllLessonProgress(c.Request.Context(), userID)
	progressMap := make(map[int64]database.LessonProgress)
	for _, p := range progressList {
		progressMap[p.CourseID] = p
	}

	result := make([]gin.H, 0)
	for _, course := range courses {
		item := gin.H{
			"id":             course.ID,
			"title":          course.Title,
			"description":    course.Description,
			"status":         course.Status,
			"source_doc_ids": course.SourceDocIds,
		}
		if p, ok := progressMap[course.ID]; ok {
			item["progress"] = gin.H{
				"current_module": p.CurrentModule,
				"completed":      p.Completed,
				"score_pct":      p.ScorePct,
			}
		}
		result = append(result, item)
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetCourseForPlay(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	userID := c.GetInt64("user_id")

	// Record streak engagement (silent, best-effort)
	_ = h.Queries.RecordStreak(c.Request.Context(), userID)

	course, err := h.Queries.GetCourseByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}
	modules, _ := h.Queries.GetModulesByCourse(c.Request.Context(), id)
	items, _ := h.Queries.GetCourseItemsByCourse(c.Request.Context(), id)

	itemMap := make(map[int64][]gin.H)
	for _, item := range items {
		mid := item.ModuleID.Int64
		itemMap[mid] = append(itemMap[mid], gin.H{
			"id":         item.ID,
			"item_type":  item.ItemType,
			"sort_order": item.SortOrder,
			"data":       json.RawMessage(item.Data),
		})
	}

	mods := make([]gin.H, 0)
	for _, m := range modules {
		modItems := itemMap[m.ID]
		if modItems == nil {
			modItems = []gin.H{}
		}
		mods = append(mods, gin.H{
			"id":          m.ID,
			"title":       m.Title,
			"description": m.Description,
			"sort_order":  m.SortOrder,
			"items":       modItems,
		})
	}

	// Get progress
	progress, _ := h.Queries.GetLessonProgress(c.Request.Context(),
		database.GetLessonProgressParams{UserID: userID, CourseID: id})

	c.JSON(http.StatusOK, gin.H{
		"id":          course.ID,
		"title":       course.Title,
		"description": course.Description,
		"status":      course.Status,
		"modules":     mods,
		"progress":    progress,
	})
}

func (h *Handler) SaveProgress(c *gin.Context) {
	var body struct {
		CourseID      int64  `json:"course_id"`
		CurrentModule int32  `json:"current_module"`
		Completed     bool   `json:"completed"`
		ScorePct      string `json:"score_pct"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetInt64("user_id")

	// Record streak engagement (silent, best-effort)
	_ = h.Queries.RecordStreak(c.Request.Context(), userID)

	pct := pgtype.Numeric{}
	pct.Scan(body.ScorePct)

	progress, err := h.Queries.UpsertLessonProgress(c.Request.Context(),
		database.UpsertLessonProgressParams{
			UserID:        userID,
			CourseID:      body.CourseID,
			CurrentModule: body.CurrentModule,
			Completed:     body.Completed,
			ScorePct:      pct,
		})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, progress)
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/courses/published", h.ListPublished)
	r.GET("/courses/:id/play", h.GetCourseForPlay)
	r.POST("/lessons/progress", h.SaveProgress)
}
