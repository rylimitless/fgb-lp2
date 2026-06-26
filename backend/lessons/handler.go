package lessons

import (
	"encoding/json"
	database "fgb-lp/database/queries"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
			"settings":       json.RawMessage(course.Settings),
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

	course, err := h.Queries.GetCourseByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}

	// Parse course settings for enforcement
	var courseSettings map[string]interface{}
	if len(course.Settings) > 0 {
		json.Unmarshal(course.Settings, &courseSettings)
	}

	// Get current progress for enforcement checks
	progress, _ := h.Queries.GetLessonProgress(c.Request.Context(),
		database.GetLessonProgressParams{UserID: userID, CourseID: id})

	// Enforce max_attempts
	if courseSettings != nil {
		if maxAttempts, ok := courseSettings["max_attempts"].(float64); ok && maxAttempts > 0 {
			attemptCount, _ := h.Queries.GetCourseAttemptCount(c.Request.Context(),
				database.GetCourseAttemptCountParams{UserID: userID, CourseID: id})
			if attemptCount >= int32(maxAttempts) {
				c.JSON(http.StatusOK, gin.H{
					"id":          course.ID,
					"title":       course.Title,
					"description": course.Description,
					"status":      course.Status,
					"modules":     []gin.H{},
					"progress":    progress,
					"settings":    json.RawMessage(course.Settings),
					"blocked":     gin.H{"reason": "max_attempts", "message": "You have reached the maximum number of attempts for this course."},
				})
				return
			}
		}

		// Enforce days_to_complete (expiry from enrollment date)
		if daysToComplete, ok := courseSettings["days_to_complete"].(float64); ok && daysToComplete > 0 {
			if progress.StartedAt.Valid {
				expiryTime := progress.StartedAt.Time.Add(time.Duration(daysToComplete) * 24 * time.Hour)
				if time.Now().After(expiryTime) && !progress.Completed {
					c.JSON(http.StatusOK, gin.H{
						"id":          course.ID,
						"title":       course.Title,
						"description": course.Description,
						"status":      course.Status,
						"modules":     []gin.H{},
						"progress":    progress,
						"settings":    json.RawMessage(course.Settings),
						"blocked":     gin.H{"reason": "expired", "message": fmt.Sprintf("This course must be completed within %.0f days of enrollment.", daysToComplete)},
					})
					return
				}
			}
		}
	}

	// Clear item progress on retake
	if c.Query("retake") == "true" {
		_ = h.Queries.DeleteItemProgress(c.Request.Context(),
			database.DeleteItemProgressParams{UserID: userID, CourseID: id})
		// Reset module-level progress too
		pct := pgtype.Numeric{}
		pct.Scan("0")
		_, _ = h.Queries.UpsertLessonProgress(c.Request.Context(),
			database.UpsertLessonProgressParams{
				UserID: userID, CourseID: id,
				CurrentModule: 0, Completed: false, ScorePct: pct,
			})
	}

	// Record streak engagement (silent, best-effort)
	_ = h.Queries.RecordStreak(c.Request.Context(), userID)

	// Create enrollment row on first access (no-op if already enrolled)
	_ = h.Queries.EnrollInCourse(c.Request.Context(),
		database.EnrollInCourseParams{UserID: userID, CourseID: id})

	modules, _ := h.Queries.GetModulesByCourse(c.Request.Context(), id)
	items, _ := h.Queries.GetCourseItemsByCourse(c.Request.Context(), id)

	// Load item-level progress
	itemProg, _ := h.Queries.GetItemProgressByCourse(c.Request.Context(),
		database.GetItemProgressByCourseParams{UserID: userID, CourseID: id})
	itemProgMap := make(map[int64]gin.H)
	for _, ip := range itemProg {
		itemProgMap[ip.ItemID] = gin.H{
			"answer":     json.RawMessage(ip.Answer),
			"is_correct": ip.IsCorrect,
		}
	}

	itemMap := make(map[int64][]gin.H)
	for _, item := range items {
		mid := item.ModuleID.Int64
		entry := gin.H{
			"id":         item.ID,
			"item_type":  item.ItemType,
			"sort_order": item.SortOrder,
			"data":       json.RawMessage(item.Data),
		}
		if saved, ok := itemProgMap[item.ID]; ok {
			entry["saved_answer"] = saved["answer"]
			entry["saved_is_correct"] = saved["is_correct"]
		}
		itemMap[mid] = append(itemMap[mid], entry)
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

	// Re-fetch progress after possible reset
	progress, _ = h.Queries.GetLessonProgress(c.Request.Context(),
		database.GetLessonProgressParams{UserID: userID, CourseID: id})

	c.JSON(http.StatusOK, gin.H{
		"id":          course.ID,
		"title":       course.Title,
		"description": course.Description,
		"status":      course.Status,
		"modules":     mods,
		"progress":    progress,
		"settings":    json.RawMessage(course.Settings),
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

func (h *Handler) SaveItemProgress(c *gin.Context) {
	var body struct {
		CourseID  int64           `json:"course_id"`
		ItemID    int64           `json:"item_id"`
		Answer    json.RawMessage `json:"answer"`
		IsCorrect *bool           `json:"is_correct"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetInt64("user_id")

	isCorrect := pgtype.Bool{Valid: false}
	if body.IsCorrect != nil {
		isCorrect = pgtype.Bool{Bool: *body.IsCorrect, Valid: true}
	}

	prog, err := h.Queries.UpsertItemProgress(c.Request.Context(),
		database.UpsertItemProgressParams{
			UserID:    userID,
			CourseID:  body.CourseID,
			ItemID:    body.ItemID,
			Answer:    body.Answer,
			IsCorrect: isCorrect,
		})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, prog)
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/courses/published", h.ListPublished)
	r.GET("/courses/:id/play", h.GetCourseForPlay)
	r.POST("/lessons/progress", h.SaveProgress)
	r.POST("/lessons/item-progress", h.SaveItemProgress)
}
