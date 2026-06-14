package content_repository

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	Pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{Pool: pool}
}

// ContentItem is the unified representation returned to the frontend.
type ContentItem struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	Description  string `json:"description,omitempty"`
	ContentType  string `json:"content_type"` // "document" or "course"
	Status       string `json:"status"`       // document: uploaded/processing/ready/failed ; course: draft/published/archived
	ReviewStatus string `json:"review_status,omitempty"`
	Approved     bool   `json:"approved"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at,omitempty"`
	// Document-specific
	TotalChunks  int32  `json:"total_chunks,omitempty"`
	ChunksDone   int32  `json:"chunks_done,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type ListResponse struct {
	Items        []ContentItem `json:"items"`
	Total        int64         `json:"total"`
	TotalDocs    int64         `json:"total_docs"`
	TotalCourses int64         `json:"total_courses"`
}

func (h *Handler) ListContent(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	search := c.Query("q")
	typeFilter := c.Query("type") // "document", "course", or "" for all

	// Count totals
	var totalDocs, totalCourses, total int64
	h.Pool.QueryRow(c.Request.Context(),
		"SELECT count(*) FROM documents").Scan(&totalDocs)
	h.Pool.QueryRow(c.Request.Context(),
		"SELECT count(*) FROM courses").Scan(&totalCourses)
	total = totalDocs + totalCourses

	// Build the union query
	query := `
		SELECT id, title, 'document' as content_type, status, approved, review_status,
		       total_chunks, chunks_done, COALESCE(error_message, '') as error_message,
		       created_at, created_at as updated_at
		FROM documents
		WHERE ($3::text = '' OR $3 = 'document')
		  AND ($4::text = '' OR title ILIKE '%' || $4 || '%')
		UNION ALL
		SELECT id, title, 'course' as content_type, status, COALESCE(approved, false) as approved,
		       COALESCE(review_status, 'pending') as review_status,
		       0, 0, '' as error_message,
		       created_at, updated_at
		FROM courses
		WHERE ($3::text = '' OR $3 = 'course')
		  AND ($4::text = '' OR title ILIKE '%' || $4 || '%' OR description ILIKE '%' || $4 || '%')
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := h.Pool.Query(c.Request.Context(), query, limit, offset, typeFilter, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	items := make([]ContentItem, 0)
	for rows.Next() {
		var item ContentItem
		var createdAt, updatedAt time.Time
		if err := rows.Scan(
			&item.ID, &item.Title, &item.ContentType, &item.Status, &item.Approved,
			&item.ReviewStatus, &item.TotalChunks, &item.ChunksDone, &item.ErrorMessage,
			&createdAt, &updatedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		item.CreatedAt = createdAt.Format(time.RFC3339)
		item.UpdatedAt = updatedAt.Format(time.RFC3339)
		items = append(items, item)
	}

	// Recalculate filtered totals
	var filteredTotal int64
	if typeFilter == "document" {
		filteredTotal = totalDocs
	} else if typeFilter == "course" {
		filteredTotal = totalCourses
	} else {
		filteredTotal = total
	}

	// If searching or filtering, we need an accurate filtered count
	if search != "" || typeFilter != "" {
		countQuery := `
			SELECT count(*) FROM (
				SELECT id FROM documents
				WHERE ($1::text = '' OR $1 = 'document')
				  AND ($2::text = '' OR title ILIKE '%' || $2 || '%')
				UNION ALL
				SELECT id FROM courses
				WHERE ($1::text = '' OR $1 = 'course')
				  AND ($2::text = '' OR title ILIKE '%' || $2 || '%' OR description ILIKE '%' || $2 || '%')
			) sub
		`
		h.Pool.QueryRow(c.Request.Context(), countQuery, typeFilter, search).Scan(&filteredTotal)
	}

	resp := ListResponse{
		Items:        items,
		Total:        filteredTotal,
		TotalDocs:    totalDocs,
		TotalCourses: totalCourses,
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) DeleteDocument(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	// Get file path for cleanup
	var filePath string
	err = h.Pool.QueryRow(c.Request.Context(),
		"SELECT file_path FROM documents WHERE id = $1", id).Scan(&filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	// Delete the DB record (cascades to chunks)
	_, err = h.Pool.Exec(c.Request.Context(),
		"DELETE FROM documents WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete document"})
		return
	}

	// Try to remove the file (best effort)
	os.Remove(filepath.Join("uploads", filePath))
	c.JSON(http.StatusOK, gin.H{"message": "Document deleted"})
}

func (h *Handler) DeleteCourse(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	_, err = h.Pool.Exec(c.Request.Context(),
		"DELETE FROM courses WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete course"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Course deleted"})
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/content-repository", h.ListContent)
	r.DELETE("/content-repository/documents/:id", h.DeleteDocument)
	r.DELETE("/content-repository/courses/:id", h.DeleteCourse)
}
