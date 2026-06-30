package learning_paths

import (
	"fgb-lp/audit"
	database "fgb-lp/database/queries"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	Queries *database.Queries
}

func NewHandler(queries *database.Queries) *Handler {
	return &Handler{Queries: queries}
}

// --- Admin routes ---

func (h *Handler) List(c *gin.Context) {
	paths, err := h.Queries.GetLearningPaths(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if paths == nil {
		paths = []database.LearningPath{}
	}
	c.JSON(http.StatusOK, paths)
}

type createBody struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
}

func (h *Handler) Create(c *gin.Context) {
	var body createBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetInt64("user_id")

	path, err := h.Queries.CreateLearningPath(c.Request.Context(), database.CreateLearningPathParams{
		Title:       body.Title,
		Description: body.Description,
		CreatedBy:   userID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	audit.Log(h.Queries, c, "learning_path:create", map[string]any{
		"path_id": path.ID,
		"title":   path.Title,
	})
	c.JSON(http.StatusCreated, path)
}

type updateBody struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var body updateBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update title/description
	if body.Title != "" || body.Description != "" {
		existing, err := h.Queries.GetLearningPathByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "learning path not found"})
			return
		}
		title := body.Title
		if title == "" {
			title = existing.Title
		}
		desc := body.Description
		if desc == "" {
			desc = existing.Description
		}
		path, err := h.Queries.UpdateLearningPath(c.Request.Context(), database.UpdateLearningPathParams{
			ID:          id,
			Title:       title,
			Description: desc,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		audit.Log(h.Queries, c, "learning_path:update", map[string]any{
			"path_id": path.ID,
			"title":   path.Title,
		})
	}

	// Update status separately if provided
	if body.Status != "" {
		path, err := h.Queries.UpdateLearningPathStatus(c.Request.Context(), database.UpdateLearningPathStatusParams{
			ID:     id,
			Status: body.Status,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		audit.Log(h.Queries, c, "learning_path:status_change", map[string]any{
			"path_id": path.ID,
			"status":  path.Status,
		})
		c.JSON(http.StatusOK, path)
		return
	}

	updated, _ := h.Queries.GetLearningPathByID(c.Request.Context(), id)
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	existing, err := h.Queries.GetLearningPathByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "learning path not found"})
		return
	}

	if err := h.Queries.DeleteLearningPath(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	audit.Log(h.Queries, c, "learning_path:delete", map[string]any{
		"path_id": existing.ID,
		"title":   existing.Title,
	})
	c.JSON(http.StatusOK, gin.H{"message": "learning path deleted"})
}

type addCourseBody struct {
	CourseID   int64 `json:"course_id" binding:"required"`
	SortOrder  int32 `json:"sort_order"`
	IsRequired *bool `json:"is_required"`
}

func (h *Handler) AddCourse(c *gin.Context) {
	pathID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var body addCourseBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	isRequired := true
	if body.IsRequired != nil {
		isRequired = *body.IsRequired
	}

	entry, err := h.Queries.AddCourseToPath(c.Request.Context(), database.AddCourseToPathParams{
		LearningPathID: pathID,
		CourseID:       body.CourseID,
		SortOrder:      body.SortOrder,
		IsRequired:     isRequired,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	audit.Log(h.Queries, c, "learning_path:add_course", map[string]any{
		"path_id":   pathID,
		"course_id": body.CourseID,
	})
	c.JSON(http.StatusCreated, entry)
}

func (h *Handler) RemoveCourse(c *gin.Context) {
	pathID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path id"})
		return
	}
	courseID, err := strconv.ParseInt(c.Param("courseId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}

	if err := h.Queries.RemoveCourseFromPath(c.Request.Context(), database.RemoveCourseFromPathParams{
		LearningPathID: pathID,
		CourseID:       courseID,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	audit.Log(h.Queries, c, "learning_path:remove_course", map[string]any{
		"path_id":   pathID,
		"course_id": courseID,
	})
	c.JSON(http.StatusOK, gin.H{"message": "course removed from path"})
}

type reorderBody struct {
	CourseID   int64 `json:"course_id" binding:"required"`
	SortOrder  int32 `json:"sort_order"`
	IsRequired bool  `json:"is_required"`
}

func (h *Handler) ReorderCourse(c *gin.Context) {
	pathID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var body reorderBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entry, err := h.Queries.UpdatePathCourseOrder(c.Request.Context(), database.UpdatePathCourseOrderParams{
		LearningPathID: pathID,
		CourseID:       body.CourseID,
		SortOrder:      body.SortOrder,
		IsRequired:     body.IsRequired,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, entry)
}

func (h *Handler) GetCourses(c *gin.Context) {
	pathID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	courses, err := h.Queries.GetPathCourses(c.Request.Context(), pathID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if courses == nil {
		courses = []database.GetPathCoursesRow{}
	}
	c.JSON(http.StatusOK, courses)
}

// --- Learner routes ---

func (h *Handler) ListPublished(c *gin.Context) {
	paths, err := h.Queries.GetPublishedLearningPaths(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if paths == nil {
		paths = []database.LearningPath{}
	}
	c.JSON(http.StatusOK, paths)
}

func (h *Handler) GetPublished(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	path, err := h.Queries.GetPublishedLearningPathByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "learning path not found"})
		return
	}

	// Get courses in this path
	courses, _ := h.Queries.GetPathCourses(c.Request.Context(), id)
	if courses == nil {
		courses = []database.GetPathCoursesRow{}
	}

	// Get user enrollment if authenticated
	userID := c.GetInt64("user_id")
	var enrollment *database.LearningPathEnrollment
	if userID > 0 {
		enr, err := h.Queries.GetLearningPathEnrollment(c.Request.Context(), database.GetLearningPathEnrollmentParams{
			UserID:         userID,
			LearningPathID: id,
		})
		if err == nil {
			enrollment = &enr
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"path":       path,
		"courses":    courses,
		"enrollment": enrollment,
	})
}

func (h *Handler) EnrollSelf(c *gin.Context) {
	pathID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Verify path is published
	path, err := h.Queries.GetPublishedLearningPathByID(c.Request.Context(), pathID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "learning path not found or not published"})
		return
	}

	userID := c.GetInt64("user_id")

	enrollment, err := h.Queries.EnrollInLearningPath(c.Request.Context(), database.EnrollInLearningPathParams{
		UserID:         userID,
		LearningPathID: pathID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	audit.Log(h.Queries, c, "learning_path:enroll", map[string]any{
		"path_id": path.ID,
		"title":   path.Title,
	})
	c.JSON(http.StatusCreated, enrollment)
}

func (h *Handler) MyEnrollments(c *gin.Context) {
	userID := c.GetInt64("user_id")

	enrollments, err := h.Queries.GetUserLearningPathEnrollments(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if enrollments == nil {
		enrollments = []database.GetUserLearningPathEnrollmentsRow{}
	}
	c.JSON(http.StatusOK, enrollments)
}

func (h *Handler) DropEnrollment(c *gin.Context) {
	enrID, err := strconv.ParseInt(c.Param("enrollmentId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid enrollment id"})
		return
	}

	enrollment, err := h.Queries.DropLearningPathEnrollment(c.Request.Context(), enrID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	audit.Log(h.Queries, c, "learning_path:drop", map[string]any{
		"enrollment_id": enrollment.ID,
	})
	c.JSON(http.StatusOK, enrollment)
}

func (h *Handler) UpdateProgress(c *gin.Context) {
	enrID, err := strconv.ParseInt(c.Param("enrollmentId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid enrollment id"})
		return
	}

	var body struct {
		ProgressPct float64 `json:"progress_pct" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var pct pgtype.Numeric
	if err := pct.Scan(strconv.FormatFloat(body.ProgressPct, 'f', 2, 64)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid progress_pct"})
		return
	}

	enrollment, err := h.Queries.UpdateLearningPathEnrollmentProgress(c.Request.Context(), database.UpdateLearningPathEnrollmentProgressParams{
		ID:          enrID,
		ProgressPct: pct,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, enrollment)
}

// RegisterRoutes wires admin endpoints.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/admin/learning-paths", h.List)
	r.POST("/admin/learning-paths", h.Create)
	r.PUT("/admin/learning-paths/:id", h.Update)
	r.DELETE("/admin/learning-paths/:id", h.Delete)
	r.POST("/admin/learning-paths/:id/courses", h.AddCourse)
	r.DELETE("/admin/learning-paths/:id/courses/:courseId", h.RemoveCourse)
	r.PUT("/admin/learning-paths/:id/reorder", h.ReorderCourse)
	r.GET("/admin/learning-paths/:id/courses", h.GetCourses)
}

// RegisterLearnerRoutes wires learner-facing endpoints.
func (h *Handler) RegisterLearnerRoutes(r *gin.RouterGroup) {
	r.GET("/learning-paths", h.ListPublished)
	r.GET("/learning-paths/:id", h.GetPublished)
	r.POST("/learning-paths/:id/enroll", h.EnrollSelf)
	r.GET("/learning-paths/my-enrollments", h.MyEnrollments)
	r.PUT("/learning-paths/enrollments/:enrollmentId/drop", h.DropEnrollment)
	r.PUT("/learning-paths/enrollments/:enrollmentId/progress", h.UpdateProgress)
}
