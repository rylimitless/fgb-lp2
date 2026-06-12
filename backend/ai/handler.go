package ai

import (
	"encoding/json"
	database "fgb-lp/database/queries"
	"fgb-lp/embeddings"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pgvector/pgvector-go"
)

const courseGenSystemPrompt = `You are an expert instructional designer. Given source material and a course topic, generate a structured course with modules that teach concepts through content blocks and then reinforce learning with interactive items.

Output ONLY valid JSON — no markdown, no explanation outside the JSON. Follow this exact structure:

{
  "title": "Course Title",
  "description": "2-3 sentence course overview",
  "modules": [
    {
      "title": "Module Title",
      "description": "What this module covers",
      "items": [
        {
          "type": "content",
          "data": {
            "body": "Paragraphs of teaching material. Explain concepts clearly with examples and key takeaways. 2-4 paragraphs of substantive content."
          }
        },
        {
          "type": "mc",
          "data": {
            "question": "Question text based on the content above?",
            "options": ["A", "B", "C", "D"],
            "correct": 2,
            "explanation": "Why this is correct"
          }
        }
      ]
    }
  ]
}

Item types and their data shapes:
- "content" (learning material): { body: "Substantive teaching content — 2-4 paragraphs covering key concepts, definitions, examples, and important points from the source material." }
- "mc" (multiple choice): { question, options: string[], correct: number (0-based index), explanation }
- "ma" (multiple answer): { question, options: string[], correct: number[], explanation }
- "tf" (true/false): { statement, answer: boolean, explanation }
- "fb" (fill-blank): { text: "sentence with ___ blanks", blanks: string[] }
- "sa" (short answer): { question, sample_answer, keywords: string[] }
- "matching": { pairs: [{left: string, right: string}] }
- "drag_sort": { items: string[] }
- "sequence": { steps: string[] }
- "scale": { question, min: number, max: number, min_label: string, max_label: string }

Rules:
- Every module MUST start with at least one "content" block that teaches the material. Do not skip this.
- Follow content with 2-4 assessment items (mc, tf, sa, etc.) to reinforce the material.
- Content blocks must be substantive — 2-4 paragraphs with key concepts, definitions, and practical examples.
- Base ALL content and questions on the provided source material. Cite facts from it.
- Do NOT use placeholders or lorem ipsum. Every field must contain real educational content.
- 3-5 modules total.
- Use "mc" most often for assessments, then "tf", "sa", "fb" where appropriate.
- Only use "matching", "drag_sort", "sequence", "scale" when the content naturally suits it.`

type Handler struct {
	Queries   *database.Queries
	LLM       *LLMClient
	EmbClient *embeddings.Client
}

func NewHandler(queries *database.Queries) *Handler {
	return &Handler{
		Queries:   queries,
		LLM:       NewLLMClient(),
		EmbClient: embeddings.NewClient(),
	}
}

// ---- JSON structures for LLM output ----

type genItem struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type genModule struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Items       []genItem `json:"items"`
}

type genCourse struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Modules     []genModule `json:"modules"`
}

// ---- Request types ----

type GenerateCourseRequest struct {
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	SourceDocIDs []int64 `json:"source_doc_ids"`
}

// ---- SSE helpers ----

type sseWriter struct {
	c       *gin.Context
	flusher http.Flusher
}

func newSSEWriter(c *gin.Context) (*sseWriter, bool) {
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return nil, false
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	return &sseWriter{c: c, flusher: flusher}, true
}

func (w *sseWriter) send(event string, data interface{}) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w.c.Writer, "event: %s\ndata: %s\n\n", event, string(jsonBytes))
	if err != nil {
		return err
	}
	w.flusher.Flush()
	return nil
}

func (w *sseWriter) sendError(msg string) {
	w.send("error", map[string]string{"message": msg})
}

// ---- Handlers ----

// GenerateCourse creates a course with AI-generated modules and items,
// streaming progress events via SSE.
func (h *Handler) GenerateCourse(c *gin.Context) {
	var req GenerateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.Description) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Description is required"})
		return
	}

	w, ok := newSSEWriter(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	// Step 1: Embed the prompt
	w.send("step", map[string]string{
		"step":   "embedding",
		"detail": "Analyzing your course description...",
	})
	embeddings, err := h.EmbClient.Embed([]string{req.Description})
	if err != nil {
		log.Printf("[ai] embed prompt: %v", err)
		w.sendError("Failed to analyze description")
		return
	}
	promptVec := pgvector.NewVector(float64ToFloat32(embeddings[0]))

	// Step 2: Search for relevant chunks
	w.send("step", map[string]string{
		"step":   "searching",
		"detail": "Searching approved documents for relevant content...",
	})
	chunks, err := h.Queries.SearchDocumentChunks(c.Request.Context(), database.SearchDocumentChunksParams{
		Embedding: promptVec,
		Limit:     15,
	})
	if err != nil {
		log.Printf("[ai] search chunks: %v", err)
		w.sendError("Failed to search documents")
		return
	}
	if len(chunks) == 0 {
		w.send("step", map[string]string{
			"step":   "searching",
			"detail": "No approved documents found. Using general knowledge...",
		})
	}

	// Step 3: Build context
	var contextBuilder strings.Builder
	for i, ch := range chunks {
		contextBuilder.WriteString(fmt.Sprintf("\n--- Source: %s (chunk %d) ---\n%s\n",
			ch.DocumentTitle, i+1, ch.Content))
	}
	context := contextBuilder.String()

	// Step 4: Call LLM
	w.send("step", map[string]string{
		"step":   "ai_writing",
		"detail": fmt.Sprintf("AI is writing your course using %d source chunks...", len(chunks)),
	})
	log.Printf("[ai] generating course: %s", req.Title)
	userPrompt := fmt.Sprintf(`Course topic: %s
Course description/idea: %s

Source material:
%s

Generate the course JSON based on the source material above.`, req.Title, req.Description, context)

	response, err := h.LLM.Chat(courseGenSystemPrompt, userPrompt)
	if err != nil {
		log.Printf("[ai] llm chat: %v", err)
		w.sendError("AI generation failed")
		return
	}

	// Step 5: Parse
	w.send("step", map[string]string{
		"step":   "parsing",
		"detail": "Structuring the course content...",
	})
	jsonStr := stripMarkdownFences(response)

	var gen genCourse
	if err := json.Unmarshal([]byte(jsonStr), &gen); err != nil {
		log.Printf("[ai] parse course json: %v\nRaw: %s", err, response[:min(len(response), 500)])
		w.sendError("Failed to parse AI response")
		return
	}

	// Build source references for the frontend
	sourceRefs := make([]gin.H, 0, len(chunks))
	for _, ch := range chunks {
		excerpt := ch.Content
		if len(excerpt) > 300 {
			excerpt = excerpt[:300] + "..."
		}
		sourceRefs = append(sourceRefs, gin.H{
			"document_id":    ch.DocumentID,
			"document_title": ch.DocumentTitle,
			"chunk_index":    ch.ChunkIndex,
			"excerpt":        excerpt,
		})
	}

	// Step 6: Store
	w.send("step", map[string]string{
		"step":   "saving",
		"detail": fmt.Sprintf("Saving course: %s (%d modules)...", gen.Title, len(gen.Modules)),
	})
	settingsMap := map[string]interface{}{
		"sources": sourceRefs,
	}
	settingsJSON, _ := json.Marshal(settingsMap)

	course, err := h.Queries.CreateCourse(c.Request.Context(), database.CreateCourseParams{
		Title:        gen.Title,
		Description:  gen.Description,
		CreatedBy:    1,
		SourceDocIds: req.SourceDocIDs,
		Settings:     settingsJSON,
	})
	if err != nil {
		log.Printf("[ai] create course: %v", err)
		w.sendError("Failed to save course")
		return
	}

	modulesResult := make([]gin.H, 0)
	for mi, mod := range gen.Modules {
		module, err := h.Queries.CreateModule(c.Request.Context(), database.CreateModuleParams{
			CourseID:    course.ID,
			Title:       mod.Title,
			Description: mod.Description,
			SortOrder:   int32(mi),
		})
		if err != nil {
			log.Printf("[ai] create module: %v", err)
			continue
		}

		itemsResult := make([]gin.H, 0)
		for ii, item := range mod.Items {
			ci, err := h.Queries.CreateCourseItem(c.Request.Context(), database.CreateCourseItemParams{
				CourseID:  course.ID,
				ModuleID:  pgtype.Int8{Int64: module.ID, Valid: true},
				ItemType:  item.Type,
				SortOrder: int32(ii),
				Data:      []byte(item.Data),
			})
			if err != nil {
				log.Printf("[ai] create item: %v", err)
				continue
			}
			itemsResult = append(itemsResult, gin.H{
				"id":         ci.ID,
				"item_type":  ci.ItemType,
				"sort_order": ci.SortOrder,
				"data":       json.RawMessage(ci.Data),
			})
		}

		modulesResult = append(modulesResult, gin.H{
			"id":          module.ID,
			"title":       module.Title,
			"description": module.Description,
			"sort_order":  module.SortOrder,
			"items":       itemsResult,
		})
	}

	// Done — send full course with sources
	w.send("done", gin.H{
		"id":             course.ID,
		"title":          course.Title,
		"description":    course.Description,
		"status":         course.Status,
		"source_doc_ids": course.SourceDocIds,
		"sources":        sourceRefs,
		"modules":        modulesResult,
	})
}

// ListCourses returns all courses.
func (h *Handler) ListCourses(c *gin.Context) {
	courses, err := h.Queries.GetCourses(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch courses"})
		return
	}
	if courses == nil {
		courses = []database.Course{}
	}
	c.JSON(http.StatusOK, courses)
}

// GetCourse returns a course with its modules and items.
func (h *Handler) GetCourse(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID"})
		return
	}

	course, err := h.Queries.GetCourseByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	modules, _ := h.Queries.GetModulesByCourse(c.Request.Context(), id)
	items, _ := h.Queries.GetCourseItemsByCourse(c.Request.Context(), id)

	// Parse sources from settings
	var sources interface{}
	if len(course.Settings) > 0 {
		var settingsMap map[string]json.RawMessage
		if json.Unmarshal(course.Settings, &settingsMap) == nil {
			if raw, ok := settingsMap["sources"]; ok {
				json.Unmarshal(raw, &sources)
			}
		}
	}

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

	modulesResult := make([]gin.H, 0)
	for _, m := range modules {
		modItems := itemMap[m.ID]
		if modItems == nil {
			modItems = []gin.H{}
		}
		modulesResult = append(modulesResult, gin.H{
			"id":          m.ID,
			"title":       m.Title,
			"description": m.Description,
			"sort_order":  m.SortOrder,
			"items":       modItems,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             course.ID,
		"title":          course.Title,
		"description":    course.Description,
		"status":         course.Status,
		"source_doc_ids": course.SourceDocIds,
		"sources":        sources,
		"modules":        modulesResult,
	})
}

// RegisterRoutes adds course generation routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/courses/generate", h.GenerateCourse)
	r.GET("/courses", h.ListCourses)
	r.GET("/courses/:id", h.GetCourse)
	r.PUT("/courses/:id/edit", h.EditCourse)
}

// EditCourse modifies an existing course based on AI-driven edit instructions.
func (h *Handler) EditCourse(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID"})
		return
	}

	var body struct {
		Instructions string `json:"instructions"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Instructions) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instructions are required"})
		return
	}

	// Fetch existing course structure
	course, err := h.Queries.GetCourseByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}
	modules, _ := h.Queries.GetModulesByCourse(c.Request.Context(), id)
	items, _ := h.Queries.GetCourseItemsByCourse(c.Request.Context(), id)

	// Build current course JSON
	itemMap := make(map[int64][]gin.H)
	for _, item := range items {
		mid := item.ModuleID.Int64
		itemMap[mid] = append(itemMap[mid], gin.H{
			"item_type":  item.ItemType,
			"sort_order": item.SortOrder,
			"data":       json.RawMessage(item.Data),
		})
	}
	modulesJSON := make([]gin.H, 0)
	for _, m := range modules {
		modItems := itemMap[m.ID]
		if modItems == nil {
			modItems = []gin.H{}
		}
		modulesJSON = append(modulesJSON, gin.H{
			"title":       m.Title,
			"description": m.Description,
			"items":       modItems,
		})
	}
	currentJSON, _ := json.Marshal(gin.H{
		"title":       course.Title,
		"description": course.Description,
		"modules":     modulesJSON,
	})

	// Build edit prompt
	editPrompt := fmt.Sprintf(`Here is an existing course in JSON format:

%s

Edit instructions: %s

Apply these edits and return the FULL modified course JSON (not just the changes). Follow the same structure.`, string(currentJSON), body.Instructions)

	log.Printf("[ai] editing course %d: %s", id, body.Instructions)
	response, err := h.LLM.Chat(courseGenSystemPrompt, editPrompt)
	if err != nil {
		log.Printf("[ai] edit llm: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI edit failed"})
		return
	}

	// Parse modified course
	jsonStr := stripMarkdownFences(response)
	var gen genCourse
	if err := json.Unmarshal([]byte(jsonStr), &gen); err != nil {
		log.Printf("[ai] parse edit json: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse AI response"})
		return
	}

	// Update course title/description
	_, err = h.Queries.UpdateCourseMeta(c.Request.Context(), database.UpdateCourseMetaParams{
		ID:          id,
		Title:       gen.Title,
		Description: gen.Description,
	})
	if err != nil {
		log.Printf("[ai] update course: %v", err)
	}

	// Replace modules and items
	h.Queries.DeleteCourseItems(c.Request.Context(), id)
	h.Queries.DeleteCourseModules(c.Request.Context(), id)

	modulesResult := make([]gin.H, 0)
	for mi, mod := range gen.Modules {
		module, err := h.Queries.CreateModule(c.Request.Context(), database.CreateModuleParams{
			CourseID:    id,
			Title:       mod.Title,
			Description: mod.Description,
			SortOrder:   int32(mi),
		})
		if err != nil {
			continue
		}
		itemsResult := make([]gin.H, 0)
		for ii, item := range mod.Items {
			ci, err := h.Queries.CreateCourseItem(c.Request.Context(), database.CreateCourseItemParams{
				CourseID:  id,
				ModuleID:  pgtype.Int8{Int64: module.ID, Valid: true},
				ItemType:  item.Type,
				SortOrder: int32(ii),
				Data:      []byte(item.Data),
			})
			if err != nil {
				continue
			}
			itemsResult = append(itemsResult, gin.H{
				"id":         ci.ID,
				"item_type":  ci.ItemType,
				"sort_order": ci.SortOrder,
				"data":       json.RawMessage(ci.Data),
			})
		}
		modulesResult = append(modulesResult, gin.H{
			"id":          module.ID,
			"title":       module.Title,
			"description": module.Description,
			"sort_order":  module.SortOrder,
			"items":       itemsResult,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          id,
		"title":       gen.Title,
		"description": gen.Description,
		"status":      course.Status,
		"modules":     modulesResult,
	})
}

// ---- helpers ----

func stripMarkdownFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
	}
	return strings.TrimSpace(s)
}

func float64ToFloat32(in []float64) []float32 {
	out := make([]float32, len(in))
	for i, v := range in {
		out[i] = float32(v)
	}
	return out
}
