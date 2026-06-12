package coach

import (
	"fgb-lp/ai"
	database "fgb-lp/database/queries"
	"fgb-lp/embeddings"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pgvector/pgvector-go"
)

const coachSystemPrompt = `You are Gia, an AI learning coach. Answer questions using ONLY the provided source material from approved documents.

Rules:
- Base your answer exclusively on the source material provided below.
- If the sources don't contain the answer, say "I don't find enough information in the approved documents to answer that."
- Keep answers clear, concise, and educational. Use bullet points when helpful.
- Cite which document(s) you used at the end of your answer, like: [Source: Document Title]
- Be friendly and encouraging — you're a learning coach.`

type Handler struct {
	Queries   *database.Queries
	LLM       *ai.LLMClient
	EmbClient *embeddings.Client
}

func NewHandler(queries *database.Queries) *Handler {
	return &Handler{
		Queries:   queries,
		LLM:       ai.NewLLMClient(),
		EmbClient: embeddings.NewClient(),
	}
}

type ChatRequest struct {
	Message string `json:"message"`
}

func (h *Handler) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Message) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message required"})
		return
	}

	emb, err := h.EmbClient.Embed([]string{req.Message})
	if err != nil {
		log.Printf("[coach] embed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process question"})
		return
	}

	chunks, err := h.Queries.SearchDocumentChunks(c.Request.Context(), database.SearchDocumentChunksParams{
		Embedding: pgvector.NewVector(float64ToFloat32(emb[0])),
		Limit:     5,
	})
	if err != nil {
		log.Printf("[coach] search: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}

	var ctxBuilder strings.Builder
	sources := make([]gin.H, 0)
	seen := make(map[string]bool)
	for _, ch := range chunks {
		ctxBuilder.WriteString(fmt.Sprintf("\n--- Document: %s ---\n%s\n", ch.DocumentTitle, ch.Content))
		if !seen[ch.DocumentTitle] {
			seen[ch.DocumentTitle] = true
			sources = append(sources, gin.H{
				"document_id":    ch.DocumentID,
				"document_title": ch.DocumentTitle,
			})
		}
	}

	if len(chunks) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"answer":  "I don't find any approved documents in the system yet. Please upload and approve some documents first.",
			"sources": []gin.H{},
		})
		return
	}

	userPrompt := fmt.Sprintf("Question: %s\n\nSource material:\n%s\n\nAnswer the question using only the sources above.", req.Message, ctxBuilder.String())
	answer, err := h.LLM.Chat(coachSystemPrompt, userPrompt)
	if err != nil {
		log.Printf("[coach] llm: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"answer":  answer,
		"sources": sources,
	})
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/coach/chat", h.Chat)
}

func float64ToFloat32(in []float64) []float32 {
	out := make([]float32, len(in))
	for i, v := range in {
		out[i] = float32(v)
	}
	return out
}
