package documents

import (
	database "fgb-lp/database/queries"
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
	title := c.PostForm("title")
	if title == "" {
		title = "Untitled"
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing file"})
		return
	}
	defer file.Close()

	ext := filepath.Ext(header.Filename)
	if ext != ".pdf" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only PDF files are supported"})
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

	doc, err := h.Queries.UpdateDocumentStatus(c.Request.Context(), database.UpdateDocumentStatusParams{
		ID:       id,
		Status:   "ready",
		Approved: pgtype.Bool{Bool: body.Approved, Valid: true},
	})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

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
	r.POST("/documents/upload", h.UploadDocument)
	r.DELETE("/documents/:id", h.DeleteDocument)
	r.PUT("/documents/:id/approve", h.ApproveDocument)
	r.GET("/documents/:id/chunks", h.GetDocumentChunks)
}
