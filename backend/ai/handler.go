package ai

import (
	"encoding/json"
	database "fgb-lp/database/queries"
	"fgb-lp/embeddings"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

// ---- System prompts ----

// Step 1: Course Outliner — produces Markdown, not JSON.
const coursePlanSystemPrompt = `You are an expert instructional designer. Given source material and a course topic, your job is to produce a high-level course plan consisting of module titles and descriptions.

CRITICAL RULES:
1. Structure your output EXACTLY like the Markdown example below. Do NOT use JSON, and do not write any introductory or concluding conversational filler.
2. Generate 1-7 modules based on how much meaningful material the source contains. If the source is short or narrow, 1-3 modules is fine. Only use 5-7 if the source is genuinely broad and deep. Do not pad with empty modules.
3. Order modules logically — build from foundational concepts to advanced applications.
4. Base ALL module topics strictly on the provided source material. Every topic must be traceable to the source.
5. Make titles and descriptions specific, academic, and substantive — not generic.

OUTPUT FORMAT:
# Course Title: [Insert Compelling Title]
[Insert a 3-4 sentence compelling course overview explaining what the learner will master.]

## Module 1: [Module Title]
**Description:** [2-3 sentences describing what this module covers and its specific learning objectives.]

## Module 2: [Module Title]
**Description:** [2-3 sentences describing what this module covers and its specific learning objectives.]`

// Step 2a: Section Lister — identifies 3-5 key sub-topics within a module.
const moduleSectionListerPrompt = `You are an expert instructional designer. Given a course module's title, description, and source material, identify 3-5 key sub-topics or sections that comprehensively break down this module's content.

Output ONLY a valid JSON array of section titles — no markdown, no other text:

["Section Title 1", "Section Title 2", "Section Title 3"]

CRITICAL RULES:
- Produce 3-5 sections that together cover all the relevant source material for this module.
- Order sections logically — foundational concepts first, then deeper material.
- Each title should be specific and substantive, not generic.
- Cover ALL key ideas from the source. Do not skip important concepts.`

// Step 2b: Section Content Writer — produces raw Markdown content for ONE specific section.
const moduleSectionWriterPrompt = `You are an expert instructional designer and university-level educator. Your primary job is to produce rich, thorough, and faithful teaching material for ONE SPECIFIC SECTION of a course module.

You are writing the raw content for this section ONLY. Do not include quiz questions, formatting code, or structural wrappers.

CRITICAL RULES:
1. Write as many paragraphs as are needed to faithfully and thoroughly cover this section's topic. Cover every key concept, definition, example, argument, relationship between ideas, and practical takeaway. Do not abbreviate, summarize, or truncate.
2. Be absolutely FAITHFUL to the information in the source. Paraphrasing is encouraged, but every fact, claim, concept, and example must accurately reflect the source. Do not invent, embellish, or add information not supported by the source.
3. Organize your writing using clear Markdown subheadings (####), bold text, and bullet points where appropriate to make it highly readable and scannable.
4. Do NOT use placeholders, TBD, or lorem ipsum.
5. Do NOT include any quiz questions, multiple choice, true/false, or assessment items. This is PURE CONTENT only.`

// Step 3: Question Generator — produces exactly 1 MCQ from a content summary.
// The full content is NOT sent to the LLM — only a summary for context.
// The JSON wrapping is done programmatically in Go code.
const questionGenPrompt = `You are an expert assessment designer. Based on the educational content summary provided, generate exactly ONE high-quality multiple-choice question.

Output ONLY valid JSON for the question data — no markdown, no other text:

{
  "question": "A meaningful multiple-choice question testing deep comprehension/application of the content — NOT trivial recall of names/dates.",
  "options": ["Option A", "Option B", "Option C", "Option D"],
  "correct": 0,
  "explanation": "A thorough 2-4 sentence explanation of why this answer is correct and why the others are fundamentally wrong."
}

CRITICAL RULES:
1. The question must test meaningful understanding of the concepts — NOT trivial fact recall.
2. All 4 options must be plausible. The incorrect options should be common misconceptions or related-but-wrong answers.
3. The explanation must be thorough — explain both why the correct answer is right AND why each wrong answer is wrong.
4. "correct" is a 0-based index into the options array.
5. Output ONLY the JSON object shown above. No wrapping, no markdown fences, no extra text.`

// EditCourse system prompt — for AI-driven course editing.
const editCourseSystemPrompt = `You are an expert instructional designer and course editor. Given an existing course in JSON format and edit instructions, produce the full modified course JSON.

Output ONLY valid JSON — no markdown. Structure:

{
  "title": "Course Title",
  "description": "Course overview paragraph(s)",
  "modules": [
    {
      "title": "Module Title",
      "description": "Module description",
      "items": [
        {"type": "content", "data": { "body": "..." }},
        {"type": "mc", "data": { "question": "...", "options": [...], "correct": 0, "explanation": "..." }}
      ]
    }
  ]
}

CRITICAL RULES:
- Apply the edit instructions faithfully while preserving the overall course structure.
- Content must remain accurate and educational.
- Keep assessment items meaningful and not trivial.
- Do not use placeholders or lorem ipsum.`

type Handler struct {
	Queries   *database.Queries
	LLM       *LLMClient
	EmbClient *embeddings.Client
	Jobs      *JobStore
	Worker    *Worker
}

func NewHandler(queries *database.Queries) *Handler {
	llm := NewLLMClient()
	emb := embeddings.NewClient()
	store := newJobStore(queries)
	worker := newWorker(store, queries, llm, emb)
	return &Handler{
		Queries:   queries,
		LLM:       llm,
		EmbClient: emb,
		Jobs:      store,
		Worker:    worker,
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

// planModule is the parsed result from Step 1 Markdown output.
type planModule struct {
	Title       string
	Description string
}

type plan struct {
	Title       string
	Description string
	Modules     []planModule
}

// ---- Request types ----

type GenerateCourseRequest struct {
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	SourceDocIDs []int64 `json:"source_doc_ids"`
}

// ---- SSE writer ----

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

// GenerateCourse enqueues a course generation job and returns immediately.
// The client should then subscribe to the job's SSE stream for progress updates.
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

	job := h.Jobs.create(req)
	h.Worker.enqueue(job)

	c.JSON(http.StatusAccepted, gin.H{
		"job_id": job.ID,
		"status": job.Status,
	})
}

// GetJobStatus returns the current state of a generation job.
func (h *Handler) GetJobStatus(c *gin.Context) {
	id := JobID(c.Param("id"))
	job := h.Jobs.get(id)
	if job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	job.mu.RLock()
	defer job.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"id":         job.ID,
		"status":     job.Status,
		"steps":      job.Steps,
		"modules":    job.Modules,
		"result":     job.Result,
		"error":      job.Error,
		"created_at": job.CreatedAt,
	})
}

// StreamJob opens an SSE connection and streams progress events for a job.
func (h *Handler) StreamJob(c *gin.Context) {
	id := JobID(c.Param("id"))
	job := h.Jobs.get(id)
	if job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	w, ok := newSSEWriter(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	sub := job.subscribe()
	defer job.unsubscribe(sub)

	// Replay any steps that already happened (catch-up)
	job.mu.RLock()
	for _, step := range job.Steps {
		w.send("step", step)
	}
	for _, mod := range job.Modules {
		w.send("module", gin.H{"module": mod})
	}
	if job.Status == "completed" {
		w.send("done", job.Result)
		job.mu.RUnlock()
		return
	}
	if job.Status == "failed" {
		w.send("error", map[string]string{"message": job.Error})
		job.mu.RUnlock()
		return
	}
	job.mu.RUnlock()

	// Stream new events as they arrive
	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-sub:
			if !ok {
				return
			}
			w.send(event.Event, event.Data)
			if event.Event == "done" || event.Event == "error" {
				return
			}
		}
	}
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

// GetCourse returns a single course with modules and items.
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

// PreviewCourse returns any course with modules and items for student-perspective preview,
// regardless of publish status.
func (h *Handler) PreviewCourse(c *gin.Context) {
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
		"title":       course.Title,
		"description": course.Description,
		"modules":     modulesResult,
	})
}

// ListActiveJobs returns all currently active generation jobs.
// The frontend uses this to discover in-progress generations across tabs.
func (h *Handler) ListActiveJobs(c *gin.Context) {
	active := h.Jobs.listActive()
	result := make([]gin.H, len(active))
	for i, job := range active {
		job.mu.RLock()
		result[i] = gin.H{
			"id":         job.ID,
			"status":     job.Status,
			"steps":      job.Steps,
			"modules":    job.Modules,
			"created_at": job.CreatedAt,
		}
		job.mu.RUnlock()
	}
	c.JSON(http.StatusOK, result)
}

// RegisterRoutes adds course generation routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/courses/generate", h.GenerateCourse)
	r.GET("/courses/generate/active", h.ListActiveJobs)
	r.GET("/courses/generate/:id", h.GetJobStatus)
	r.GET("/courses/generate/:id/stream", h.StreamJob)
	r.GET("/courses", h.ListCourses)
	r.GET("/courses/:id", h.GetCourse)
	r.GET("/courses/:id/preview", h.PreviewCourse)
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
	response, err := h.LLM.Chat(editCourseSystemPrompt, editPrompt)
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

// parsePlanMarkdown extracts a course plan from Step 1's Markdown output.
//
// Expected format:
//
//	# Course Title: [Title]
//	[Description paragraph(s)]
//
//	## Module 1: [Module Title]
//	**Description:** [Description]
//
//	## Module 2: [Module Title]
//	**Description:** [Description]
func parsePlanMarkdown(md string) plan {
	result := plan{}

	// Split on "## Module" to find module boundaries.
	// Everything before the first "## Module" is the course header.
	moduleRe := regexp.MustCompile(`(?m)^## Module \d+:\s*`)
	locs := moduleRe.FindAllStringIndex(md, -1)

	var headerSection string
	var moduleSections []string

	if len(locs) == 0 {
		// Fallback: treat the whole thing as header
		headerSection = md
	} else {
		headerSection = md[:locs[0][0]]
		for i, loc := range locs {
			end := len(md)
			if i+1 < len(locs) {
				end = locs[i+1][0]
			}
			moduleSections = append(moduleSections, md[loc[0]:end])
		}
	}

	// Parse course title from header
	titleRe := regexp.MustCompile(`(?m)^#\s+(?:Course Title:\s*)?(.+)$`)
	if match := titleRe.FindStringSubmatch(headerSection); match != nil {
		result.Title = strings.TrimSpace(match[1])
	}

	// Parse course description: everything in header that's not the title line
	descLines := strings.Split(headerSection, "\n")
	var descBuilder strings.Builder
	for _, line := range descLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		descBuilder.WriteString(trimmed)
		descBuilder.WriteString("\n")
	}
	result.Description = strings.TrimSpace(descBuilder.String())

	// Parse each module
	for _, sec := range moduleSections {
		pm := planModule{}

		// Extract module title from the "## Module N: Title" line
		modTitleRe := regexp.MustCompile(`^## Module \d+:\s*(.+)$`)
		lines := strings.Split(sec, "\n")
		for _, line := range lines {
			if match := modTitleRe.FindStringSubmatch(strings.TrimSpace(line)); match != nil {
				pm.Title = strings.TrimSpace(match[1])
				break
			}
		}

		// Extract description from **Description:** line
		descRe := regexp.MustCompile(`\*\*Description:\*\*\s*(.+)`)
		for _, line := range lines {
			if match := descRe.FindStringSubmatch(line); match != nil {
				pm.Description = strings.TrimSpace(match[1])
				break
			}
		}

		// Also try without bold markers (some LLMs drop the **)
		if pm.Description == "" {
			plainDescRe := regexp.MustCompile(`Description:\s*(.+)`)
			for _, line := range lines {
				if match := plainDescRe.FindStringSubmatch(line); match != nil {
					pm.Description = strings.TrimSpace(match[1])
					break
				}
			}
		}

		if pm.Title != "" {
			result.Modules = append(result.Modules, pm)
		}
	}

	return result
}

func float64ToFloat32(in []float64) []float32 {
	out := make([]float32, len(in))
	for i, v := range in {
		out[i] = float32(v)
	}
	return out
}
