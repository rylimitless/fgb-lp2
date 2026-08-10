package documents

import (
	"fgb-lp/audit"
	database "fgb-lp/database/queries"
	"fgb-lp/middlewares"
	"fgb-lp/worker"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	Queries   *database.Queries
	UploadDir string
}

func NewHandler(queries *database.Queries, uploadDir string) *Handler {
	return &Handler{
		Queries:   queries,
		UploadDir: uploadDir,
	}
}

func (h *Handler) ListDocuments(c *gin.Context) {
	docs, err := h.Queries.GetDocuments(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch documents"})
		return
	}
	if docs == nil {
		docs = []database.Document{}
	}
	c.JSON(http.StatusOK, docs)
}

func (h *Handler) UploadDocument(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing file"})
		return
	}
	defer file.Close()

	title := c.PostForm("title")
	if title == "" {
		title = header.Filename
	}

	if !worker.IsSupportedExtension(header.Filename) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Unsupported file type — accepts PDF, TXT, and Markdown.",
		})
		return
	}

	filename := worker.GenerateFilePath(header.Filename)
	destPath := filepath.Join(h.UploadDir, filename)

	out, err := os.Create(destPath)
	if err != nil {
		log.Printf("[documents] failed to create file %s: %v", destPath, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		log.Printf("[documents] failed to write file %s: %v", destPath, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	doc, err := h.Queries.InsertDocument(c.Request.Context(), database.InsertDocumentParams{
		Title:    title,
		FilePath: filename,
	})
	if err != nil {
		log.Printf("[documents] failed to insert document: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create document"})
		return
	}

	audit.Log(h.Queries, c, "document_uploaded", map[string]any{
		"document_id": doc.ID,
		"title":       doc.Title,
		"filename":    header.Filename,
	})

	c.JSON(http.StatusCreated, doc)
}

func (h *Handler) DeleteDocument(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	doc, err := h.Queries.GetDocumentByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	filePath := filepath.Join(h.UploadDir, doc.FilePath)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		log.Printf("[documents] failed to remove file %s: %v", filePath, err)
	}

	if err := h.Queries.DeleteDocument(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete document"})
		return
	}

	audit.Log(h.Queries, c, "document_deleted", map[string]any{
		"document_id": id,
		"title":       doc.Title,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Document deleted"})
}

func (h *Handler) ApproveDocument(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	var body struct {
		Approved bool `json:"approved"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Rejected documents must not be marked 'ready' — that would let them be
	// used for coaching/retrieval. 'rejected' keeps them out of the active set.
	status := "ready"
	if !body.Approved {
		status = "rejected"
	}

	doc, err := h.Queries.UpdateDocumentStatus(c.Request.Context(), database.UpdateDocumentStatusParams{
		ID:       id,
		Status:   status,
		Approved: pgtype.Bool{Bool: body.Approved, Valid: true},
	})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	audit.Log(h.Queries, c, "document_approved", map[string]any{
		"document_id": id,
		"title":       doc.Title,
		"approved":    body.Approved,
	})

	c.JSON(http.StatusOK, doc)
}

func (h *Handler) GetDocumentChunks(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}
	chunks, err := h.Queries.GetDocumentChunks(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chunks"})
		return
	}
	if chunks == nil {
		chunks = []database.DocumentChunk{}
	}
	c.JSON(http.StatusOK, chunks)
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/documents", h.ListDocuments)
	r.POST("/documents/upload", middlewares.WrapRequireRole(h.UploadDocument, "content creator"))
	r.DELETE("/documents/:id", middlewares.WrapRequireRole(h.DeleteDocument, "content creator"))
	r.PUT("/documents/:id/approve", middlewares.WrapRequireRole(h.ApproveDocument, "approver"))
	r.GET("/documents/:id/chunks", h.GetDocumentChunks)
}
