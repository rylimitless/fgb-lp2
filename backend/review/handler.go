package review

import (
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

func (h *Handler) ListDocuments(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	docs, err := h.Queries.GetPendingReviewDocuments(c.Request.Context(),
		database.GetPendingReviewDocumentsParams{Limit: int32(limit), Offset: int32(offset)})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if docs == nil {
		docs = []database.Document{}
	}
	total, _ := h.Queries.CountPendingReviewDocuments(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"items": docs, "total": total})
}

func (h *Handler) ListCourses(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	courses, err := h.Queries.GetPendingReviewCourses(c.Request.Context(),
		database.GetPendingReviewCoursesParams{Limit: int32(limit), Offset: int32(offset)})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if courses == nil {
		courses = []database.Course{}
	}
	total, _ := h.Queries.CountPendingReviewCourses(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"items": courses, "total": total})
}

func (h *Handler) ReviewDocument(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	var body struct {
		ReviewStatus string `json:"review_status"`
		ReviewNotes  string `json:"review_notes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	approved := body.ReviewStatus == "approved"
	doc, err := h.Queries.UpdateDocumentReview(c.Request.Context(), database.UpdateDocumentReviewParams{
		ID:           id,
		ReviewStatus: pgtype.Text{String: body.ReviewStatus, Valid: true},
		ReviewNotes:  pgtype.Text{String: body.ReviewNotes, Valid: true},
		Approved:     pgtype.Bool{Bool: approved, Valid: true},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, doc)
}

func (h *Handler) ReviewCourse(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	var body struct {
		ReviewStatus string `json:"review_status"`
		ReviewNotes  string `json:"review_notes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	approved := body.ReviewStatus == "approved"
	course, err := h.Queries.UpdateCourseReview(c.Request.Context(), database.UpdateCourseReviewParams{
		ID:           id,
		ReviewStatus: pgtype.Text{String: body.ReviewStatus, Valid: true},
		ReviewNotes:  pgtype.Text{String: body.ReviewNotes, Valid: true},
		Approved:     pgtype.Bool{Bool: approved, Valid: true},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, course)
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/review/documents", h.ListDocuments)
	r.GET("/review/courses", h.ListCourses)
	r.PUT("/review/documents/:id", h.ReviewDocument)
	r.PUT("/review/courses/:id", h.ReviewCourse)
}
