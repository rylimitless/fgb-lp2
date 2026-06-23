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
	"golang.org/x/sync/errgroup"
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

// recoverFromDB marks any interrupted jobs as failed on startup.
// We intentionally DO NOT resume interrupted generations: re-running
// processJob from scratch created duplicate courses on every restart,
// wasting LLM credits and producing runaway module counts. If a job
// was interrupted, the user can manually re-trigger generation.
func (s *JobStore) recoverFromDB() {
	ctx := context.Background()

	// Mark ALL previously pending/running jobs as failed.
	// We intentionally DO NOT resume interrupted generations: re-running
	// processJob from scratch created duplicate courses on every restart,
	// wasting LLM credits and producing runaway module counts. If a job
	// was interrupted, the user can manually re-trigger generation.
	rows, err := s.db.ListActiveGenerationJobs(ctx)
	if err != nil {
		log.Printf("[jobstore] list active jobs: %v", err)
		return
	}
	for _, row := range rows {
		msg := "cancelled: server restarted during generation"
		if row.CourseID != nil {
			msg = fmt.Sprintf("interrupted during generation; partial course #%d was kept", *row.CourseID)
		}
		log.Printf("[jobstore] cancelling interrupted job %s (%s)", row.ID, msg)
		s.db.FailGenerationJob(ctx, row.ID, msg)
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
		store.recoverFromDB()
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

	// --- Embed the course description (used for broad outline retrieval) ---
	job.addStep("embedding", "Analyzing your course description...")
	embeddings, err := w.emb.Embed([]string{req.Description})
	if err != nil {
		log.Printf("[worker] embed: %v", err)
		w.failJob(job, "Failed to analyze description")
		return
	}
	promptVec := pgvector.NewVector(float64ToFloat32(embeddings[0]))

	// --- Broad retrieval so the outline sees the document's full scope ---
	job.addStep("searching", "Searching approved documents for relevant content...")
	chunks, err := w.queries.SearchDocumentChunks(context.Background(), database.SearchDocumentChunksParams{
		Embedding: promptVec,
		Limit:     outlineChunkLimit,
	})
	if err != nil {
		log.Printf("[worker] search: %v", err)
		w.failJob(job, "Failed to search documents")
		return
	}
	if len(chunks) == 0 {
		job.addStep("searching", "No approved documents found. Using general knowledge...")
	}

	sourceContext := buildChunkContext(chunks)

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

	// Hard cap to prevent runaway generation from an overly ambitious LLM plan.
	const maxModules = 25
	if len(coursePlan.Modules) > maxModules {
		log.Printf("[worker] job %s step1: clamping %d modules to %d", job.ID, len(coursePlan.Modules), maxModules)
		coursePlan.Modules = coursePlan.Modules[:maxModules]
	}
	log.Printf("[worker] job %s step1: %d modules", job.ID, len(coursePlan.Modules))

	// --- Build source refs (course-level, from broad retrieval) ---
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
	// STEP 2/3: Generate each module IN PARALLEL.
	// Each module retrieves its OWN source chunks (per-module retrieval), so
	// a large document is actually covered instead of every module rewriting
	// the same handful of chunks. modulesResult is indexed by module position
	// so final ordering is preserved regardless of completion order.
	// =====================================================================
	totalModules := len(coursePlan.Modules)
	modulesResult := make([]gin.H, totalModules)

	g, gctx := errgroup.WithContext(context.Background())
	g.SetLimit(moduleConcurrency)

	for mi, modPlan := range coursePlan.Modules {
		mi, modPlan := mi, modPlan
		g.Go(func() error {
			modResult, err := w.generateModule(gctx, job, courseID, coursePlan, mi, modPlan, totalModules)
			if err != nil {
				return err
			}
			modulesResult[mi] = modResult
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		w.failJob(job, err.Error())
		return
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

// --- Course generation tuning constants ---
const (
	outlineChunkLimit = 200 // broad retrieval so the outline reflects the full document scope
	moduleChunkLimit  = 60  // focused per-module retrieval
	moduleConcurrency = 3   // parallel module generation (bounded to respect LLM rate limits)
)

// buildChunkContext renders retrieved chunks into a single source-material string for an LLM prompt.
func buildChunkContext(chunks []database.SearchDocumentChunksRow) string {
	var b strings.Builder
	for i, ch := range chunks {
		b.WriteString(fmt.Sprintf("\n--- Source: %s (chunk %d) ---\n%s\n",
			ch.DocumentTitle, i+1, ch.Content))
	}
	return b.String()
}

// generateModule runs the per-module pipeline (retrieval → sections → content → questions)
// for a single module. Each section gets its content written, then 1-3 questions are
// generated about that section. Items are stored interleaved: content, q1, q2, content, q3...
func (w *Worker) generateModule(ctx context.Context, job *GenerationJob, courseID int64, coursePlan plan, mi int, modPlan planModule, totalModules int) (gin.H, error) {
	moduleLabel := fmt.Sprintf("%d/%d", mi+1, totalModules)

	// --- Per-module retrieval ---
	job.addStep("retrieving", fmt.Sprintf("Finding source material for Module %s — %s...", moduleLabel, modPlan.Title))
	modEmb, err := w.emb.Embed([]string{modPlan.Title + ". " + modPlan.Description})
	if err != nil {
		log.Printf("[worker] module %s embed: %v", moduleLabel, err)
		return nil, fmt.Errorf("Failed to retrieve material for module %d", mi+1)
	}
	modVec := pgvector.NewVector(float64ToFloat32(modEmb[0]))

	modChunks, err := w.queries.SearchDocumentChunks(ctx, database.SearchDocumentChunksParams{
		Embedding: modVec,
		Limit:     moduleChunkLimit,
	})
	if err != nil {
		log.Printf("[worker] module %s search: %v", moduleLabel, err)
		return nil, fmt.Errorf("Failed to retrieve material for module %d", mi+1)
	}
	moduleContext := buildChunkContext(modChunks)

	module, err := w.queries.CreateModule(ctx, database.CreateModuleParams{
		CourseID:    courseID,
		Title:       modPlan.Title,
		Description: modPlan.Description,
		SortOrder:   int32(mi),
	})
	if err != nil {
		log.Printf("[worker] create module %d: %v", mi+1, err)
		return nil, fmt.Errorf("Failed to create module %d", mi+1)
	}

	// --- Step 2a: Section Lister ---
	job.addStep("writing", fmt.Sprintf("Analyzing structure for Module %s — %s...", moduleLabel, modPlan.Title))

	sectionPrompt := fmt.Sprintf(`Course: %s — %s
Module: "%s" — %s
Source Material (use only parts relevant to this module):
%s
List 3-5 key sections that comprehensively break down this module's content.`,
		coursePlan.Title, coursePlan.Description, modPlan.Title, modPlan.Description, moduleContext)

	sectionResp, err := w.llm.Chat(moduleSectionListerPrompt, sectionPrompt)
	if err != nil {
		log.Printf("[worker] step2a (module %d): %v", mi+1, err)
		return nil, fmt.Errorf("Step 2a failed for module %d", mi+1)
	}

	sectionJSON := stripMarkdownFences(sectionResp)
	var sectionTitles []string
	if err := json.Unmarshal([]byte(sectionJSON), &sectionTitles); err != nil || len(sectionTitles) == 0 {
		sectionTitles = []string{modPlan.Title}
	}

	// --- Step 2b+3: Per-section content + interleaved questions ---
	// Each section gets its content written, then 1-3 questions generated about
	// that section. Items are stored interleaved: content, q1, q2, content, q3, ...
	// This replaces the old approach of batching all content then all 8 questions.
	itemsResult := make([]gin.H, 0)
	sortOrder := int32(0)

	allowedTypes := map[string]bool{
		"mc": true, "ma": true, "tf": true, "fb": true, "sa": true,
		"matching": true, "drag_sort": true, "hotspot": true,
		"sequence": true, "scale": true,
	}

	for si, secTitle := range sectionTitles {
		// --- Write content for this section ---
		job.addStep("writing", fmt.Sprintf("Writing section %d/%d of Module %s — %s...",
			si+1, len(sectionTitles), moduleLabel, secTitle))

		contentPrompt := fmt.Sprintf(`Course: %s — %s
Module: "%s" — %s
Section: "%s"
Source Material (use only parts relevant to this section):
%s
Write exhaustive, faithful teaching content for this section.`,
			coursePlan.Title, coursePlan.Description, modPlan.Title, modPlan.Description, secTitle, moduleContext)

		secContent, err := w.llm.Chat(moduleSectionWriterPrompt, contentPrompt)
		if err != nil {
			log.Printf("[worker] content (m%d s%d): %v", mi+1, si+1, err)
			return nil, fmt.Errorf("Content generation failed for module %d section %d", mi+1, si+1)
		}

		// Store content item
		sectionBody := strings.TrimSpace(secContent)
		if len(sectionTitles) > 1 {
			sectionBody = fmt.Sprintf("### %s\n\n%s", secTitle, sectionBody)
		}
		contentData, err := json.Marshal(map[string]string{"body": sectionBody})
		if err != nil {
			return nil, fmt.Errorf("Content encoding error for module %d section %d", mi+1, si+1)
		}
		ciContent, err := w.queries.CreateCourseItem(ctx, database.CreateCourseItemParams{
			CourseID:  courseID,
			ModuleID:  pgtype.Int8{Int64: module.ID, Valid: true},
			ItemType:  "content",
			SortOrder: sortOrder,
			Data:      []byte(contentData),
		})
		if err == nil {
			itemsResult = append(itemsResult, gin.H{
				"id": ciContent.ID, "item_type": "content", "sort_order": sortOrder,
				"data": json.RawMessage(contentData),
			})
			sortOrder++
		} else {
			log.Printf("[worker] create content item (module %d section %d): %v", mi+1, si+1, err)
		}

		// --- Generate 1-3 questions about this section ---
		job.addStep("writing", fmt.Sprintf("Creating questions for section %d/%d of Module %s...",
			si+1, len(sectionTitles), moduleLabel))

		contentSummary := sectionBody
		if len(contentSummary) > 2000 {
			contentSummary = contentSummary[:2000]
		}

		questionPrompt := fmt.Sprintf(`Section: "%s"

Content:
%s

Generate 1-3 assessment items that test understanding of THIS section.`, secTitle, contentSummary)

		questionResp, err := w.llm.Chat(perSectionQuestionPrompt, questionPrompt)
		if err != nil {
			log.Printf("[worker] questions (m%d s%d): %v", mi+1, si+1, err)
			// Non-fatal: continue to next section even if questions fail
			continue
		}

		questionJSON := stripMarkdownFences(questionResp)
		var questions []genItem
		if err := json.Unmarshal([]byte(questionJSON), &questions); err != nil {
			log.Printf("[worker] parse questions (m%d s%d): %v — skipping questions", mi+1, si+1, err)
			continue
		}

		for _, q := range questions {
			qType := strings.ToLower(strings.TrimSpace(q.Type))
			if !allowedTypes[qType] {
				log.Printf("[worker] unknown item_type %q (m%d s%d) — defaulting to mc", q.Type, mi+1, si+1)
				qType = "mc"
			}
			ci, err := w.queries.CreateCourseItem(ctx, database.CreateCourseItemParams{
				CourseID:  courseID,
				ModuleID:  pgtype.Int8{Int64: module.ID, Valid: true},
				ItemType:  qType,
				SortOrder: sortOrder,
				Data:      []byte(q.Data),
			})
			if err != nil {
				log.Printf("[worker] create question (m%d s%d): %v", mi+1, si+1, err)
				continue
			}
			itemsResult = append(itemsResult, gin.H{
				"id": ci.ID, "item_type": qType, "sort_order": sortOrder,
				"data": q.Data,
			})
			sortOrder++
		}
	}

	modResult := gin.H{
		"id":          module.ID,
		"title":       modPlan.Title,
		"description": modPlan.Description,
		"sort_order":  module.SortOrder,
		"items":       itemsResult,
	}
	job.addModule(modResult, moduleLabel)
	return modResult, nil
}

func float64ToFloat32Job(in []float64) []float32 {
	return float64ToFloat32(in)
}
