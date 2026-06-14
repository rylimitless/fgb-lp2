package review

import (
	"context"
	database "fgb-lp/database/queries"
	"fmt"
	"net/http"
	"strconv"
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

// reviewerIdentity returns "Name (Role)" for the currently authenticated user.
func (h *Handler) reviewerIdentity(c *gin.Context) string {
	userID, exists := c.Get("user_id")
	if !exists {
		return "Unknown"
	}
	user, err := h.Queries.GetUserByID(c.Request.Context(), userID.(int64))
	if err != nil {
		return "Unknown"
	}
	if user.Name != "" {
		return fmt.Sprintf("%s (%s)", user.Name, user.Role)
	}
	return user.Role
}

// sendDocNotification creates a notification for the document owner about a review action.
func (h *Handler) sendDocNotification(doc database.Document, status string) {
	if !doc.UploadedBy.Valid {
		return
	}
	ownerID := doc.UploadedBy.Int64

	var title, message string
	switch status {
	case "approved":
		title = "Document approved"
		message = fmt.Sprintf("Your document \"%s\" has been approved.", doc.Title)
	case "changes_requested":
		title = "Changes requested"
		message = fmt.Sprintf("Changes were requested for your document \"%s\".", doc.Title)
	case "rejected":
		title = "Document rejected"
		message = fmt.Sprintf("Your document \"%s\" was rejected.", doc.Title)
	default:
		return
	}

	h.Queries.CreateNotification(context.Background(), database.CreateNotificationParams{
		UserID:  pgtype.Int8{Int64: ownerID, Valid: true},
		Title:   title,
		Message: message,
		Link:    "/content-repository",
	})
}

// sendCourseNotification creates a notification for the course owner about a review action.
func (h *Handler) sendCourseNotification(course database.Course, status string) {
	ownerID := course.CreatedBy

	var title, message string
	switch status {
	case "approved":
		title = "Course approved"
		message = fmt.Sprintf("Your course \"%s\" has been approved and published.", course.Title)
	case "changes_requested":
		title = "Changes requested"
		message = fmt.Sprintf("Changes were requested for your course \"%s\".", course.Title)
	case "rejected":
		title = "Course rejected"
		message = fmt.Sprintf("Your course \"%s\" was rejected.", course.Title)
	default:
		return
	}

	h.Queries.CreateNotification(context.Background(), database.CreateNotificationParams{
		UserID:  pgtype.Int8{Int64: ownerID, Valid: true},
		Title:   title,
		Message: message,
		Link:    "/content-repository",
	})
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

	// Build conversation-style notes: prepend new note with timestamp + identity
	finalNotes := body.ReviewNotes
	if body.ReviewNotes != "" {
		ts := time.Now().Format("2006-01-02 15:04")
		who := h.reviewerIdentity(c)
		prefix := fmt.Sprintf("[%s %s] %s", ts, who, body.ReviewNotes)

		// Fetch existing notes to preserve history
		existing, err := h.Queries.GetDocumentByID(c.Request.Context(), id)
		if err == nil && existing.ReviewNotes.String != "" {
			finalNotes = prefix + "\n---\n" + existing.ReviewNotes.String
		} else {
			finalNotes = prefix
		}
	} else {
		// Preserve existing notes when no new notes provided (e.g. direct approve)
		existing, err := h.Queries.GetDocumentByID(c.Request.Context(), id)
		if err == nil {
			finalNotes = existing.ReviewNotes.String
		}
	}

	approved := body.ReviewStatus == "approved"
	doc, err := h.Queries.UpdateDocumentReview(c.Request.Context(), database.UpdateDocumentReviewParams{
		ID:           id,
		ReviewStatus: pgtype.Text{String: body.ReviewStatus, Valid: true},
		ReviewNotes:  pgtype.Text{String: finalNotes, Valid: true},
		Approved:     pgtype.Bool{Bool: approved, Valid: true},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Notify the document owner
	h.sendDocNotification(doc, body.ReviewStatus)

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

	// Build conversation-style notes: prepend new note with timestamp + identity
	finalNotes := body.ReviewNotes
	if body.ReviewNotes != "" {
		ts := time.Now().Format("2006-01-02 15:04")
		who := h.reviewerIdentity(c)
		prefix := fmt.Sprintf("[%s %s] %s", ts, who, body.ReviewNotes)

		// Fetch existing notes to preserve history
		existing, err := h.Queries.GetCourseByID(c.Request.Context(), id)
		if err == nil && existing.ReviewNotes.String != "" {
			finalNotes = prefix + "\n---\n" + existing.ReviewNotes.String
		} else {
			finalNotes = prefix
		}
	} else {
		// Preserve existing notes when no new notes provided (e.g. direct approve)
		existing, err := h.Queries.GetCourseByID(c.Request.Context(), id)
		if err == nil {
			finalNotes = existing.ReviewNotes.String
		}
	}

	approved := body.ReviewStatus == "approved"
	var approvedBy pgtype.Int8
	if approved {
		userID, exists := c.Get("user_id")
		if exists {
			approvedBy = pgtype.Int8{Int64: userID.(int64), Valid: true}
		}
	}
	course, err := h.Queries.UpdateCourseReview(c.Request.Context(), database.UpdateCourseReviewParams{
		ID:           id,
		ReviewStatus: pgtype.Text{String: body.ReviewStatus, Valid: true},
		ReviewNotes:  pgtype.Text{String: finalNotes, Valid: true},
		Approved:     pgtype.Bool{Bool: approved, Valid: true},
		ApprovedBy:   approvedBy,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Notify the course owner
	h.sendCourseNotification(course, body.ReviewStatus)

	c.JSON(http.StatusOK, course)
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/review/documents", h.ListDocuments)
	r.GET("/review/courses", h.ListCourses)
	r.PUT("/review/documents/:id", h.ReviewDocument)
	r.PUT("/review/courses/:id", h.ReviewCourse)
}
