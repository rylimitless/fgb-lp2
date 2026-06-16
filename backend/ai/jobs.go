package ai

import (
	"context"
	"encoding/json"
	"fgb-lp/audit"
	database "fgb-lp/database/queries"
	"fgb-lp/embeddings"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pgvector/pgvector-go"
)

// ---- Job types ----

type JobID string

type SSEEvent struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

type subscriber chan SSEEvent

// GenerationJob tracks the full lifecycle of a course generation.
// Steps/Modules are mirrored to DB via persistProgress() for crash recovery.
type GenerationJob struct {
	ID        JobID                 `json:"id"`
	Status    string                `json:"status"`
	Request   GenerateCourseRequest `json:"-"`
	Steps     []map[string]string   `json:"steps"`
	Modules   []gin.H               `json:"modules"`
	Result    gin.H                 `json:"result,omitempty"`
	Error     string                `json:"error,omitempty"`
	CreatedAt time.Time             `json:"created_at"`

	subs map[subscriber]struct{}
	mu   sync.RWMutex

	// DB-backed state: used to persist progress
	db       *database.Queries
	courseID *int64
}

func (j *GenerationJob) subscribe() subscriber {
	j.mu.Lock()
	defer j.mu.Unlock()
	ch := make(subscriber, 64)
	if j.subs == nil {
		j.subs = make(map[subscriber]struct{})
	}
	j.subs[ch] = struct{}{}
	return ch
}

func (j *GenerationJob) unsubscribe(ch subscriber) {
	j.mu.Lock()
	defer j.mu.Unlock()
	delete(j.subs, ch)
}

func (j *GenerationJob) broadcast(event SSEEvent) {
	j.mu.RLock()
	defer j.mu.RUnlock()
	for ch := range j.subs {
		select {
		case ch <- event:
		default:
		}
	}
}

func (j *GenerationJob) addStep(step, detail string) {
	j.mu.Lock()
	j.Steps = append(j.Steps, map[string]string{"step": step, "detail": detail})
	j.mu.Unlock()
	j.broadcast(SSEEvent{Event: "step", Data: map[string]string{"step": step, "detail": detail}})
	j.persistProgress()
}

func (j *GenerationJob) addModule(mod gin.H, progress string) {
	j.mu.Lock()
	j.Modules = append(j.Modules, mod)
	j.mu.Unlock()
	j.broadcast(SSEEvent{Event: "module", Data: gin.H{"module": mod, "progress": progress}})
	j.persistProgress()
}

// persistProgress writes current steps/modules to the DB for crash recovery.
// Errors are logged but not fatal — progress is best-effort.
func (j *GenerationJob) persistProgress() {
	if j.db == nil {
		return
	}
	stepsJSON, _ := json.Marshal(j.Steps)
	modulesJSON, _ := json.Marshal(j.Modules)
	if err := j.db.UpdateGenerationJobProgress(context.Background(), string(j.ID), stepsJSON, modulesJSON); err != nil {
		log.Printf("[job] persistProgress %s: %v", j.ID, err)
	}
}

// ---- JobStore (DB-backed) ----

type JobStore struct {
	db          *database.Queries
	inFlightMap map[JobID]*GenerationJob // in-memory index of active jobs
	mu          sync.RWMutex
}

func newJobStore(db *database.Queries) *JobStore {
	return &JobStore{
		db:          db,
		inFlightMap: make(map[JobID]*GenerationJob),
	}
}

func (s *JobStore) create(req GenerateCourseRequest) *GenerationJob {
	job := &GenerationJob{
		ID:        JobID(uuid.New().String()),
		Status:    "pending",
		Request:   req,
		CreatedAt: time.Now(),
		db:        s.db,
	}
	// Persist to DB
	reqJSON, _ := json.Marshal(req)
	if err := s.db.CreateGenerationJob(context.Background(), string(job.ID), reqJSON); err != nil {
		log.Printf("[jobstore] create DB write failed: %v", err)
	}
	s.mu.Lock()
	s.inFlightMap[job.ID] = job
	s.mu.Unlock()
	return job
}

func (s *JobStore) get(id JobID) *GenerationJob {
	s.mu.RLock()
	job := s.inFlightMap[id]
	s.mu.RUnlock()
	if job != nil {
		return job
	}
	// Try DB for completed/failed jobs
	row, err := s.db.GetGenerationJob(context.Background(), string(id))
	if err != nil {
		return nil
	}
	return dbRowToJob(row, s.db)
}

func (s *JobStore) listActive() []*GenerationJob {
	s.mu.RLock()
	active := make([]*GenerationJob, 0, len(s.inFlightMap))
	for _, job := range s.inFlightMap {
		job.mu.RLock()
		status := job.Status
		job.mu.RUnlock()
		if status == "pending" || status == "running" {
			active = append(active, job)
		}
	}
	s.mu.RUnlock()
	return active
}

// recoverFromDB fetches pending jobs from DB and emits them to the worker on startup.
func (s *JobStore) recoverFromDB(worker *Worker) {
	ctx := context.Background()

	// Reset any running jobs that were left over from a crash
	if err := s.db.ResetStaleJobs(ctx); err != nil {
		log.Printf("[jobstore] reset stale jobs: %v", err)
	}

	rows, err := s.db.ListActiveGenerationJobs(ctx)
	if err != nil {
		log.Printf("[jobstore] list active jobs: %v", err)
		return
	}

	for _, row := range rows {
		job := dbRowToJob(row, s.db)
		s.mu.Lock()
		s.inFlightMap[job.ID] = job
		s.mu.Unlock()
		log.Printf("[jobstore] recovered job %s (%s)", job.ID, job.Status)
		worker.enqueue(job)
	}
}

func dbRowToJob(row *database.GenerationJobRow, db *database.Queries) *GenerationJob {
	job := &GenerationJob{
		ID:     JobID(row.ID),
		Status: row.Status,
		db:     db,
	}
	if row.CourseID != nil {
		job.courseID = row.CourseID
	}
	json.Unmarshal(row.Request, &job.Request)
	json.Unmarshal(row.Steps, &job.Steps)
	json.Unmarshal(row.Modules, &job.Modules)
	json.Unmarshal(row.Result, &job.Result)
	job.Error = row.Error
	// Steps/Modules are already set from DB
	if job.Steps == nil {
		job.Steps = []map[string]string{}
	}
	if job.Modules == nil {
		job.Modules = []gin.H{}
	}
	return job
}

// ---- Worker ----

type Worker struct {
	store   *JobStore
	queries *database.Queries
	llm     *LLMClient
	emb     *embeddings.Client
	queue   chan *GenerationJob
}

func newWorker(store *JobStore, queries *database.Queries, llm *LLMClient, emb *embeddings.Client) *Worker {
	w := &Worker{
		store:   store,
		queries: queries,
		llm:     llm,
		emb:     emb,
		queue:   make(chan *GenerationJob, 32),
	}
	go w.run()
	// Recover any jobs that were pending/running when the server last stopped
	go func() {
		time.Sleep(500 * time.Millisecond) // let everything initialize
		store.recoverFromDB(w)
	}()
	return w
}

func (w *Worker) enqueue(job *GenerationJob) {
	w.queue <- job
}

func (w *Worker) run() {
	for job := range w.queue {
		w.processJob(job)
	}
}

func (w *Worker) failJob(job *GenerationJob, msg string) {
	log.Printf("[worker] job %s FAILED: %s", job.ID, msg)
	job.mu.Lock()
	job.Status = "failed"
	job.Error = msg
	job.mu.Unlock()
	w.queries.FailGenerationJob(context.Background(), string(job.ID), msg)
	job.broadcast(SSEEvent{Event: "error", Data: map[string]string{"message": msg}})
}

func (w *Worker) completeJob(job *GenerationJob, result gin.H, courseID int64) {
	log.Printf("[worker] job %s COMPLETED", job.ID)
	job.mu.Lock()
	job.Status = "completed"
	job.Result = result
	job.courseID = &courseID
	title := job.Request.Title
	job.mu.Unlock()
	resultJSON, _ := json.Marshal(result)
	w.queries.CompleteGenerationJob(context.Background(), string(job.ID), resultJSON, courseID)
	job.broadcast(SSEEvent{Event: "done", Data: result})

	// Audit log the course creation
	audit.Log(w.queries, nil, "course_created", map[string]any{
		"course_id": courseID,
		"title":     title,
	})
	// Notify all users that a new course is ready
	notifyAll(w.queries,
		"Course generation complete",
		fmt.Sprintf("\"%s\" is ready for review.", title),
		"/review-queue",
	)
}

// processJob runs the full 3-step generation pipeline in a background goroutine.
func (w *Worker) processJob(job *GenerationJob) {
	job.mu.Lock()
	job.Status = "running"
	reqTitle := job.Request.Title
	job.mu.Unlock()
	w.queries.StartGenerationJob(context.Background(), string(job.ID))

	// Notify admins that generation has started
	notifyAdmins(w.queries,
		"Course generation started",
		fmt.Sprintf("AI is generating \"%s\".", reqTitle),
		"/ai-content-generator",
	)

	defer func() {
		if r := recover(); r != nil {
			log.Printf("[worker] PANIC in job %s: %v", job.ID, r)
			w.failJob(job, fmt.Sprintf("internal panic: %v", r))
		}
	}()

	req := job.Request
	if req.CreatedBy == 0 {
		w.failJob(job, "Invalid user session")
		return
	}

	// --- Embed ---
	job.addStep("embedding", "Analyzing your course description...")
	embeddings, err := w.emb.Embed([]string{req.Description})
	if err != nil {
		log.Printf("[worker] embed: %v", err)
		w.failJob(job, "Failed to analyze description")
		return
	}
	promptVec := pgvector.NewVector(float64ToFloat32(embeddings[0]))

	// --- Search ---
	job.addStep("searching", "Searching approved documents for relevant content...")
	chunks, err := w.queries.SearchDocumentChunks(context.Background(), database.SearchDocumentChunksParams{
		Embedding: promptVec,
		Limit:     40,
	})
	if err != nil {
		log.Printf("[worker] search: %v", err)
		w.failJob(job, "Failed to search documents")
		return
	}
	if len(chunks) == 0 {
		job.addStep("searching", "No approved documents found. Using general knowledge...")
	}

	var contextBuilder strings.Builder
	for i, ch := range chunks {
		contextBuilder.WriteString(fmt.Sprintf("\n--- Source: %s (chunk %d) ---\n%s\n",
			ch.DocumentTitle, i+1, ch.Content))
	}
	sourceContext := contextBuilder.String()

	// =====================================================================
	// STEP 1: COURSE OUTLINER
	// =====================================================================
	job.addStep("planning", "Step 1/3: Creating course outline and module structure...")
	log.Printf("[worker] job %s STEP 1 (outliner): %s", job.ID, req.Title)

	sourceCharCount := len(sourceContext)
	chunkCount := len(chunks)

	planPrompt := fmt.Sprintf(`Course topic: %s
Course description/idea: %s

Source material statistics: %d total characters across %d chunks.

Source material:
%s

Generate a Markdown course plan PROPORTIONAL to the source material size shown above.`,
		req.Title, req.Description, sourceCharCount, chunkCount, sourceContext)

	planResp, err := w.llm.Chat(coursePlanSystemPrompt, planPrompt)
	if err != nil {
		log.Printf("[worker] step1 outliner: %v", err)
		w.failJob(job, "Step 1 failed: outline generation error")
		return
	}

	coursePlan := parsePlanMarkdown(planResp)
	if len(coursePlan.Modules) == 0 {
		w.failJob(job, "Step 1 failed: no modules found")
		return
	}
	log.Printf("[worker] job %s step1: %d modules", job.ID, len(coursePlan.Modules))

	// --- Build source refs ---
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

	settingsMap := map[string]interface{}{"sources": sourceRefs}
	settingsJSON, _ := json.Marshal(settingsMap)

	// --- Create course record ---
	job.addStep("saving", fmt.Sprintf("Saving course outline: %s (%d modules)...",
		coursePlan.Title, len(coursePlan.Modules)))

	course, err := w.queries.CreateCourse(context.Background(), database.CreateCourseParams{
		Title:        coursePlan.Title,
		Description:  coursePlan.Description,
		CreatedBy:    req.CreatedBy,
		SourceDocIds: req.SourceDocIDs,
		Settings:     settingsJSON,
	})
	if err != nil {
		log.Printf("[worker] create course: %v", err)
		w.failJob(job, "Failed to save course")
		return
	}
	courseID := course.ID

	// =====================================================================
	// For each module: STEP 2a → 2b → 3
	// =====================================================================
	modulesResult := make([]gin.H, 0, len(coursePlan.Modules))

	for mi, modPlan := range coursePlan.Modules {
		module, err := w.queries.CreateModule(context.Background(), database.CreateModuleParams{
			CourseID:    courseID,
			Title:       modPlan.Title,
			Description: modPlan.Description,
			SortOrder:   int32(mi),
		})
		if err != nil {
			log.Printf("[worker] create module %d: %v", mi+1, err)
			w.failJob(job, fmt.Sprintf("Failed to create module %d", mi+1))
			return
		}

		// --- STEP 2a: Section Lister ---
		job.addStep("writing", fmt.Sprintf("Step 2/3: Analyzing structure for Module %d/%d — %s...",
			mi+1, len(coursePlan.Modules), modPlan.Title))

		sectionPrompt := fmt.Sprintf(`Course: %s — %s
Module: "%s" — %s
Source Material (use only parts relevant to this module):
%s
List 3-5 key sections that comprehensively break down this module's content.`,
			coursePlan.Title, coursePlan.Description, modPlan.Title, modPlan.Description, sourceContext)

		sectionResp, err := w.llm.Chat(moduleSectionListerPrompt, sectionPrompt)
		if err != nil {
			log.Printf("[worker] step2a (module %d): %v", mi+1, err)
			w.failJob(job, fmt.Sprintf("Step 2a failed for module %d", mi+1))
			return
		}

		sectionJSON := stripMarkdownFences(sectionResp)
		var sectionTitles []string
		if err := json.Unmarshal([]byte(sectionJSON), &sectionTitles); err != nil || len(sectionTitles) == 0 {
			sectionTitles = []string{modPlan.Title}
		}

		// --- STEP 2b: Per-section content writer ---
		var allContent strings.Builder
		for si, secTitle := range sectionTitles {
			job.addStep("writing", fmt.Sprintf("Step 2/3: Writing section %d/%d of Module %d/%d — %s...",
				si+1, len(sectionTitles), mi+1, len(coursePlan.Modules), secTitle))

			contentPrompt := fmt.Sprintf(`Course: %s — %s
Module: "%s" — %s
Section: "%s"
Source Material (use only parts relevant to this section):
%s
Write exhaustive, faithful teaching content for this section.`,
				coursePlan.Title, coursePlan.Description, modPlan.Title, modPlan.Description, secTitle, sourceContext)

			secContent, err := w.llm.Chat(moduleSectionWriterPrompt, contentPrompt)
			if err != nil {
				log.Printf("[worker] step2b (m%d s%d): %v", mi+1, si+1, err)
				w.failJob(job, fmt.Sprintf("Step 2b failed for module %d section %d", mi+1, si+1))
				return
			}

			if len(sectionTitles) > 1 {
				allContent.WriteString(fmt.Sprintf("\n### %s\n\n", secTitle))
			}
			allContent.WriteString(strings.TrimSpace(secContent))
			allContent.WriteString("\n\n")
		}
		rawContent := strings.TrimSpace(allContent.String())

		// --- STEP 3: Question generator ---
		job.addStep("packaging", fmt.Sprintf("Step 3/3: Generating assessment for Module %d/%d...",
			mi+1, len(coursePlan.Modules)))

		contentSummary := rawContent
		if len(contentSummary) > 2500 {
			contentSummary = contentSummary[:2500] + "\n\n[... content continues for " +
				fmt.Sprintf("%d", len(rawContent)-2500) + " more characters ...]"
		}

		questionPrompt := fmt.Sprintf(`Module: "%s" — %s

Content Summary (full content is %d characters total):
%s

Based on the concepts covered above, generate exactly 8 assessment items as a JSON array, using these formats in order: mc, ma, tf, fb, matching, drag_sort, hotspot, sa.`,
			modPlan.Title, modPlan.Description, len(rawContent), contentSummary)

		questionResp, err := w.llm.Chat(questionGenPrompt, questionPrompt)
		if err != nil {
			log.Printf("[worker] step3 (module %d): %v", mi+1, err)
			w.failJob(job, fmt.Sprintf("Step 3 failed for module %d", mi+1))
			return
		}

		questionJSON := stripMarkdownFences(questionResp)
		var questions []genItem
		if err := json.Unmarshal([]byte(questionJSON), &questions); err != nil {
			log.Printf("[worker] step3 parse questions (module %d): %v", mi+1, err)
			repairPrompt := fmt.Sprintf(`The previous response was invalid JSON. Repair it into ONLY a valid JSON array of exactly 8 assessment items using these types in order: mc, ma, tf, fb, matching, drag_sort, hotspot, sa.

Invalid response:
%s

Output ONLY the repaired JSON array.`, questionResp)
			repairedResp, repairErr := w.llm.Chat(questionGenPrompt, repairPrompt)
			if repairErr != nil {
				log.Printf("[worker] step3 repair questions (module %d): %v", mi+1, repairErr)
				w.failJob(job, fmt.Sprintf("Step 3 failed: bad question JSON for module %d", mi+1))
				return
			}
			questionJSON = stripMarkdownFences(repairedResp)
			if err := json.Unmarshal([]byte(questionJSON), &questions); err != nil {
				log.Printf("[worker] step3 repaired JSON still invalid (module %d): %v", mi+1, err)
				w.failJob(job, fmt.Sprintf("Step 3 failed: bad question JSON for module %d", mi+1))
				return
			}
		}
		if len(questions) == 0 {
			log.Printf("[worker] step3 empty question array (module %d)", mi+1)
			w.failJob(job, fmt.Sprintf("Step 3 failed: no questions for module %d", mi+1))
			return
		}

		contentData, err := json.Marshal(map[string]string{"body": rawContent})
		if err != nil {
			w.failJob(job, fmt.Sprintf("Content encoding error for module %d", mi+1))
			return
		}

		itemsResult := make([]gin.H, 0, 1+len(questions))
		ciContent, err := w.queries.CreateCourseItem(context.Background(), database.CreateCourseItemParams{
			CourseID:  courseID,
			ModuleID:  pgtype.Int8{Int64: module.ID, Valid: true},
			ItemType:  "content",
			SortOrder: 0,
			Data:      []byte(contentData),
		})
		if err == nil {
			itemsResult = append(itemsResult, gin.H{
				"id": ciContent.ID, "item_type": "content", "sort_order": 0,
				"data": json.RawMessage(contentData),
			})
		} else {
			log.Printf("[worker] create content item (module %d): %v", mi+1, err)
		}

		// Allowed item types per the course_items check constraint.
		allowedTypes := map[string]bool{
			"mc": true, "ma": true, "tf": true, "fb": true, "sa": true,
			"matching": true, "drag_sort": true, "hotspot": true,
			"sequence": true, "scale": true,
		}

		for qi, q := range questions {
			qType := strings.ToLower(strings.TrimSpace(q.Type))
			if !allowedTypes[qType] {
				log.Printf("[worker] step3 unknown item_type %q (module %d, q %d) — defaulting to mc", q.Type, mi+1, qi+1)
				qType = "mc"
			}
			sort := int32(qi + 1)
			ci, err := w.queries.CreateCourseItem(context.Background(), database.CreateCourseItemParams{
				CourseID:  courseID,
				ModuleID:  pgtype.Int8{Int64: module.ID, Valid: true},
				ItemType:  qType,
				SortOrder: sort,
				Data:      []byte(q.Data),
			})
			if err != nil {
				log.Printf("[worker] create question item (module %d, q %d): %v", mi+1, qi+1, err)
				continue
			}
			itemsResult = append(itemsResult, gin.H{
				"id": ci.ID, "item_type": qType, "sort_order": sort,
				"data": q.Data,
			})
		}

		modResult := gin.H{
			"id":          module.ID,
			"title":       modPlan.Title,
			"description": modPlan.Description,
			"sort_order":  module.SortOrder,
			"items":       itemsResult,
		}
		modulesResult = append(modulesResult, modResult)
		job.addModule(modResult, fmt.Sprintf("%d/%d", mi+1, len(coursePlan.Modules)))
	}

	result := gin.H{
		"id":             courseID,
		"title":          coursePlan.Title,
		"description":    coursePlan.Description,
		"status":         course.Status,
		"source_doc_ids": course.SourceDocIds,
		"sources":        sourceRefs,
		"modules":        modulesResult,
	}
	w.completeJob(job, result, courseID)
}

func float64ToFloat32Job(in []float64) []float32 {
	return float64ToFloat32(in)
}
