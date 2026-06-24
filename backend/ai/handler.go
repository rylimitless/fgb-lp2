package ai

import (
	"encoding/json"
	database "fgb-lp/database/queries"
	"fgb-lp/embeddings"
	"fgb-lp/middlewares"
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
2. The number of modules MUST be proportional to the source material's actual size and depth. Use this scale:
   - Under 1,000 chars of source → 1 module
   - 1,000–8,000 chars → 2-3 modules
   - 8,000–30,000 chars → 4-6 modules
   - 30,000–80,000 chars → 7-12 modules
   - 80,000–200,000 chars → 13-20 modules
   - Over 200,000 chars → up to 25 modules
   Each module should target ~2,000-4,000 words of finished teaching content (~1-2 hours of learner effort). Do NOT split thin content into multiple modules, and do NOT cram a large source into a handful of overstuffed modules — if the source is large, more focused modules is correct.
3. Order modules logically — build from foundational concepts to advanced applications.
4. Base ALL module topics strictly on the provided source material. Every topic must be directly traceable to the source. If there isn't enough material to justify a distinct module, don't create one.
5. Make titles and descriptions specific, academic, and substantive — not generic.
6. The course overview should reflect the ACTUAL scope of the source — if the source is narrow, the overview should be 1-2 sentences, not inflated.

OUTPUT FORMAT:
# Course Title: [Insert Compelling Title]
[Insert a course overview explaining what the learner will master — length proportional to source scope.]

## Module 1: [Module Title]
**Description:** [2-3 sentences describing what this module covers and its specific learning objectives.]

## Module 2: [Module Title]
**Description:** [2-3 sentences describing what this module covers and its specific learning objectives.]`

// Step 2a: Section Lister — identifies 1-5 key sub-topics within a module.
const moduleSectionListerPrompt = `You are an expert instructional designer. Given a course module's title, description, and source material, identify key sub-topics or sections that comprehensively break down this module's content.

Output ONLY a valid JSON array of section titles — no markdown, no other text:

["Section Title 1", "Section Title 2", "Section Title 3"]

CRITICAL RULES:
- The number of sections MUST match the depth of the available source material for this module. Produce 1-5 sections — use fewer when the source is thin, more when it's rich.
- If the source material is very short (only a few paragraphs), 1-2 sections is perfectly appropriate.
- Order sections logically — foundational concepts first, then deeper material.
- Each title should be specific and substantive, not generic.
- Cover ALL key ideas from the source. Do not skip important concepts.
- Do NOT create sections that aren't supported by the source — do not invent topics to fill a quota.`

// Step 2b: Section Content Writer — produces raw Markdown content for ONE specific section.
const moduleSectionWriterPrompt = `You are an expert instructional designer and university-level educator. Your primary job is to produce faithful teaching material for ONE SPECIFIC SECTION of a course module.

You are writing the raw content for this section ONLY. Do not include quiz questions, formatting code, or structural wrappers.

CRITICAL RULES:
1. Match your output length to the source material available for this section. If the source provides only a few sentences on this topic, write 1-2 concise paragraphs. If the source is rich and detailed, you may write longer. NEVER stretch thin content into long passages — conciseness is valuable.
2. Be absolutely FAITHFUL to the information in the source. Paraphrasing is encouraged, but every fact, claim, concept, and example must accurately reflect the source. Do NOT invent, embellish, or add information not supported by the source.
3. Organize your writing using clear Markdown subheadings (####), bold text, and bullet points where appropriate to make it highly readable and scannable.
4. Do NOT use placeholders, TBD, or lorem ipsum.
5. Do NOT include any quiz questions, multiple choice, true/false, or assessment items. This is PURE CONTENT only.
6. If the source material for this section is very thin, it is better to be short and accurate than long and fabricated.`

// Step 3: Per-Section Question Generator — produces 1-3 assessment items
// that test the content JUST covered in one section. Used inside the per-section
// loop so questions are interleaved with content, not batched at the end.
const perSectionQuestionPrompt = `You are an expert assessment designer. Given ONE section of educational content, generate 1-3 high-quality assessment items that test understanding of THIS section specifically.

CRITICAL RULES:
1. Generate questions ONLY about the content in this specific section — do not draw from other topics.
2. Pick question types that BEST fit the section material. Choose from: mc (multiple choice), ma (multiple answer), tf (true/false), fb (fill-in-the-blank), sa (short answer). Matching, drag_sort, hotspot, and sequence are also allowed if the content naturally suits them.
3. Vary the types — don't use the same format for all questions.
4. Every question MUST be answerable strictly from the section content.
5. Every question MUST include an "explanation" field explaining the correct answer(s).
6. MC: exactly 4 plausible options with distractions that are common misconceptions; "correct" is a 0-based index.
7. MA: 4-6 options with 2-3 correct; "correct" is an array of 0-based indices.
8. TF: "statement" + "answer" (boolean).
9. FB: "text" with "___" blanks + "blanks" array of answers.
10. SA: "question" + "sample_answer".
11. Content must be substantive — not trivial recall of names or dates.
12. Do not use placeholders, TBD, or lorem ipsum.

Output ONLY a valid JSON array — no markdown, no surrounding text:

[
  {
    "type": "mc",
    "data": { "question": "...", "options": ["...","...","...","..."], "correct": 0, "explanation": "..." }
  },
  {
    "type": "tf",
    "data": { "statement": "...", "answer": true, "explanation": "..." }
  }
]`

// Step 3 (legacy): Question Generator — produces a batch of 8 mixed-format items
// from a content summary. The full content is NOT sent to the LLM — only a summary
// for context. The JSON wrapping is done programmatically in Go code.
const questionGenPrompt = `You are an expert assessment designer. Based on the educational content summary provided, generate EXACTLY 8 high-quality assessment items with VARIED formats.

The 8 items MUST use these exact formats, in this order:
1. Multiple choice (mc) — single correct answer
2. Multiple answer (ma) — multiple correct answers
3. True / false (tf)
4. Fill in the blank (fb)
5. Matching (matching)
6. Ordering / sequence (drag_sort)
7. Hotspot / identify area (hotspot)
8. Short answer (sa)

Output ONLY a valid JSON array — no markdown, no other text:

[
  {
    "type": "mc",
    "data": {
      "question": "...",
      "options": ["...","...","...","..."],
      "correct": 0,
      "explanation": "Why the correct answer is right and the others are wrong."
    }
  },
  {
    "type": "ma",
    "data": {
      "question": "Select all that apply.",
      "options": ["...","...","...","...","..."],
      "correct": [0, 2],
      "explanation": "Why these answers are correct and the others are not."
    }
  },
  {
    "type": "tf",
    "data": {
      "statement": "A substantive statement based directly on the source.",
      "answer": true,
      "explanation": "Why the statement is true (or false)."
    }
  },
  {
    "type": "fb",
    "data": {
      "text": "A sentence with one or more ___ to fill in.",
      "blanks": ["expected answer 1", "expected answer 2"],
      "explanation": "Why those are the correct fills."
    }
  },
  {
    "type": "matching",
    "data": {
      "question": "Match each concept with the correct definition or implication.",
      "pairs": [
        { "left": "Concept A", "right": "Definition or implication A" },
        { "left": "Concept B", "right": "Definition or implication B" },
        { "left": "Concept C", "right": "Definition or implication C" },
        { "left": "Concept D", "right": "Definition or implication D" }
      ],
      "explanation": "Why these pairings are correct."
    }
  },
  {
    "type": "drag_sort",
    "data": {
      "question": "Put these steps in the correct order.",
      "items": ["First step", "Second step", "Third step", "Fourth step"],
      "explanation": "Why this is the correct sequence."
    }
  },
  {
    "type": "hotspot",
    "data": {
      "question": "Select the area that best represents the correct concept.",
      "image": "/brand/questions/hotspot-cyber-risk.png",
      "regions": [
        { "label": "Correct region", "x": 42, "y": 54, "correct": true },
        { "label": "Distractor 1", "x": 25, "y": 35, "correct": false },
        { "label": "Distractor 2", "x": 68, "y": 42, "correct": false },
        { "label": "Distractor 3", "x": 55, "y": 75, "correct": false }
      ],
      "explanation": "Why the correct region represents the concept."
    }
  },
  {
    "type": "sa",
    "data": {
      "question": "An open-ended question requiring a 2-4 sentence answer.",
      "sample_answer": "An exemplary answer demonstrating the depth expected.",
      "explanation": "What a strong answer should cover."
    }
  }
]

CRITICAL RULES:
1. Every question MUST test meaningful understanding — never trivial recall of names or dates.
2. Every question MUST be answerable strictly from the source content; do not invent facts.
3. MC: exactly 4 plausible options; "correct" is a 0-based index; wrong options should be common misconceptions.
4. MA: 4-6 options with 2-3 correct; "correct" is an array of 0-based indices.
5. TF: state a clear claim that is unambiguously true or false based on the source.
6. FB: use literally three underscores "___" for each blank in "text"; "blanks" lists the answers in order; 1-3 blanks.
7. MATCHING: use 3-5 pairs; each left and right must be short, unambiguous, and source-grounded.
8. DRAG_SORT: use 3-5 ordered items; "items" must be in the correct order; the frontend will shuffle them.
9. HOTSPOT: use image "/brand/questions/hotspot-cyber-risk.png" unless a different existing asset is clearly more relevant from this allow-list: "/brand/pathways/compliance.png", "/brand/pathways/risk-credit.png", "/brand/pathways/customer-service.png", "/brand/pathways/cybersecurity.png", "/brand/pathways/banking-foundations.png"; include 3-4 labeled regions with exactly one correct region; x/y are percentages from 0-100 and may be approximate.
10. SA: include a substantive "sample_answer" demonstrating expected depth.
11. Every item MUST include an "explanation" field.
12. Output ONLY the JSON array of exactly 8 objects. No prose, no markdown fences.`

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

// EditItem system prompt — for AI-driven editing of a single course item.
const editItemSystemPrompt = `You are an expert instructional designer and content editor. Given a single course item in JSON format and edit instructions, produce the modified version of JUST that item.

Output ONLY valid JSON — no markdown, no explanation. Keep the same item_type and data structure:

- "content": { "data": { "body": "..." } }
- "mc": { "data": { "question": "...", "options": [...], "correct": <index>, "explanation": "..." } }
- "ma": { "data": { "question": "...", "options": [...], "correct": [<indices>], "explanation": "..." } }
- "tf": { "data": { "statement": "...", "answer": <bool>, "explanation": "..." } }
- "fb": { "data": { "text": "...", "blanks": [...] } }
- "sa": { "data": { "question": "...", "sample_answer": "..." } }
- "matching": { "data": { "question": "...", "pairs": [{"left":"...","right":"..."}] } }
- "drag_sort": { "data": { "question": "...", "items": [...] } }
- "sequence": { "data": { "question": "...", "steps": [...] } }
- "hotspot": { "data": { "question": "...", "image": "...", "regions": [{"label":"...","x":<pct>,"y":<pct>,"correct":<bool>}] } }

CRITICAL RULES:
- Apply the edit instructions faithfully to this single item.
- Keep the same item_type — do not change it.
- Maintain the same data structure/fields appropriate to the item_type.
- Content must remain accurate, educational, and substantive.
- Do not use placeholders or lorem ipsum.
- Output ONLY the JSON object — no surrounding text or markdown fences.`

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
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	SourceDocIDs  []int64  `json:"source_doc_ids"`
	QuestionTypes []string `json:"question_types,omitempty"`
	CreatedBy     int64    `json:"created_by,omitempty"`
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
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	createdBy, ok := userID.(int64)
	if !ok || createdBy == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user session"})
		return
	}
	req.CreatedBy = createdBy

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
		"approved_by":    course.ApprovedBy,
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

// CancelGeneration cancels an in-progress course generation job.
func (h *Handler) CancelGeneration(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}
	if h.Jobs.CancelJob(JobID(id)) {
		c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found or already completed"})
	}
}

// RegisterRoutes adds course generation routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/courses/generate", middlewares.WrapRequireRole(h.GenerateCourse, "content creator"))
	r.GET("/courses/generate/active", h.ListActiveJobs)
	r.POST("/courses/generate/:id/cancel", middlewares.WrapRequireRole(h.CancelGeneration, "content creator"))
	r.GET("/courses/generate/:id", h.GetJobStatus)
	r.GET("/courses/generate/:id/stream", h.StreamJob)
	r.GET("/courses", h.ListCourses)
	r.GET("/courses/:id", h.GetCourse)
	r.GET("/courses/:id/preview", h.PreviewCourse)
	r.PUT("/courses/:id/edit", middlewares.WrapRequireRole(h.EditCourse, "content creator"))
	r.POST("/items/:itemId/ai-edit", middlewares.WrapRequireRole(h.AiEditItem, "content creator"))
	r.PUT("/items/:itemId", middlewares.WrapRequireRole(h.UpdateItemData, "content creator"))
	r.DELETE("/items/:itemId", middlewares.WrapRequireRole(h.DeleteItem, "content creator"))
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

// AiEditItem returns an AI-suggested edit for a single course item.
// This does NOT persist to the database — the frontend shows a preview
// and the user must accept it via UpdateItemData.
func (h *Handler) AiEditItem(c *gin.Context) {
	itemID, err := strconv.ParseInt(c.Param("itemId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	var body struct {
		Instructions string `json:"instructions"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Instructions) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instructions are required"})
		return
	}

	// Fetch the current item
	item, err := h.Queries.GetCourseItemByID(c.Request.Context(), itemID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	// Build the item JSON for the LLM
	currentItem := gin.H{
		"item_type": item.ItemType,
		"data":      json.RawMessage(item.Data),
	}
	currentJSON, _ := json.Marshal(currentItem)

	editPrompt := fmt.Sprintf(`Here is a course item in JSON format:

%s

Edit instructions: %s

Apply these edits and return ONLY the modified JSON for this single item.`, string(currentJSON), body.Instructions)

	log.Printf("[ai] editing item %d: %s", itemID, body.Instructions)
	response, err := h.LLM.Chat(editItemSystemPrompt, editPrompt)
	if err != nil {
		log.Printf("[ai] edit item llm: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI edit failed"})
		return
	}

	jsonStr := stripMarkdownFences(response)
	var suggested genItem
	if err := json.Unmarshal([]byte(jsonStr), &suggested); err != nil {
		log.Printf("[ai] parse edit item json: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse AI response"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"item_id":        item.ID,
		"item_type":      item.ItemType,
		"current_data":   json.RawMessage(item.Data),
		"suggested_type": suggested.Type,
		"suggested_data": suggested.Data,
	})
}

// UpdateItemData persists a modified item's data to the database.
func (h *Handler) UpdateItemData(c *gin.Context) {
	itemID, err := strconv.ParseInt(c.Param("itemId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	var body struct {
		Data json.RawMessage `json:"data"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.Data) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data is required"})
		return
	}

	// Verify item exists
	_, err = h.Queries.GetCourseItemByID(c.Request.Context(), itemID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	updated, err := h.Queries.UpdateCourseItemData(c.Request.Context(), database.UpdateCourseItemDataParams{
		ID:   itemID,
		Data: []byte(body.Data),
	})
	if err != nil {
		log.Printf("[ai] update item data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         updated.ID,
		"item_type":  updated.ItemType,
		"sort_order": updated.SortOrder,
		"data":       json.RawMessage(updated.Data),
	})
}

// DeleteItem removes a course item from the database and renumbers its siblings.
func (h *Handler) DeleteItem(c *gin.Context) {
	itemID, err := strconv.ParseInt(c.Param("itemId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	// Fetch the item so we know which module to renumber
	item, err := h.Queries.GetCourseItemByID(c.Request.Context(), itemID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	if err := h.Queries.DeleteCourseItemByID(c.Request.Context(), itemID); err != nil {
		log.Printf("[ai] delete item: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete item"})
		return
	}

	// Renumber remaining items in the same module so sort_order stays contiguous
	if item.ModuleID.Valid {
		siblings, _ := h.Queries.GetCourseItemsByModule(c.Request.Context(), item.ModuleID)
		for i, sib := range siblings {
			if sib.SortOrder != int32(i) {
				h.Queries.UpdateCourseItemModule(c.Request.Context(), database.UpdateCourseItemModuleParams{
					ID:        sib.ID,
					ModuleID:  sib.ModuleID,
					SortOrder: int32(i),
				})
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"id": itemID, "deleted": true})
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
