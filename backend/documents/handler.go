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
)

type Handler struct {
	Queries   *database.Queries
	UploadDir string
}

// NewHandler creates a documents handler.
func NewHandler(queries *database.Queries, uploadDir string) *Handler {
	return &Handler{
		Queries:   queries,
		UploadDir: uploadDir,
	}
}

// ListDocuments returns all documents ordered by created_at desc.
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

// UploadDocument accepts a multipart form with "file" (PDF) and optional "title".
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

	// Validate extension
	ext := filepath.Ext(header.Filename)
	if ext != ".pdf" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only PDF files are supported"})
		return
	}

	// Generate unique filename
	filename := worker.GenerateFilePath(header.Filename)
	destPath := filepath.Join(h.UploadDir, filename)

	// Save file to disk
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

	// Insert into database
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

// DeleteDocument deletes a document, its chunks (cascaded), and the file on disk.
func (h *Handler) DeleteDocument(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	// Look up the document to get the file path
	doc, err := h.Queries.GetDocumentByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	// Delete file from disk (best-effort)
	filePath := filepath.Join(h.UploadDir, doc.FilePath)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		log.Printf("[documents] failed to remove file %s: %v", filePath, err)
	}

	// Delete from DB (cascades to document_chunks)
	if err := h.Queries.DeleteDocument(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete document"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Document deleted"})
}

// RegisterRoutes adds document routes to a gin.RouterGroup.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/documents", h.ListDocuments)
	r.POST("/documents/upload", h.UploadDocument)
	r.DELETE("/documents/:id", h.DeleteDocument)
}
