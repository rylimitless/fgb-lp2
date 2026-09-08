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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
	"golang.org/x/sync/errgroup"
)

// maxQuestionsFromSettings extracts the optional hard question cap stored in
// a course's settings jsonb (set when the author requested a fixed question
// count). Returns 0 when unset.
func maxQuestionsFromSettings(settings []byte) int {
	if len(settings) == 0 {
		return 0
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(settings, &m) != nil {
		return 0
	}
	raw, ok := m["max_questions"]
	if !ok {
		return 0
	}
	var n int
	if json.Unmarshal(raw, &n) != nil {
		return 0
	}
	if n < 0 {
		return 0
	}
	return n
}

func moduleQuestionBudget(total, moduleIndex, moduleCount int) int {
	if total <= 0 || moduleCount <= 0 || moduleIndex < 0 || moduleIndex >= moduleCount {
		return 0
	}
	budget := total / moduleCount
	if moduleIndex < total%moduleCount {
		budget++
	}
	return budget
}

func generatedConceptCount(items []database.CourseItemWithGroup) int {
	groups := make(map[string]struct{})
	count := 0
	for _, item := range items {
		if item.ItemType == "content" {
			continue
		}
		if item.QuestionGroupID.Valid {
			if _, seen := groups[item.QuestionGroupID.String]; !seen {
				groups[item.QuestionGroupID.String] = struct{}{}
				count++
			}
			continue
		}
		count++
	}
	return count
}

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
	Stage     JobStage              `json:"stage"`
	Request   GenerateCourseRequest `json:"-"`
	Steps     []map[string]string   `json:"steps"`
	Modules   []gin.H               `json:"modules"`
	Result    gin.H                 `json:"result,omitempty"`
	Error     string                `json:"error,omitempty"`
	CreatedAt time.Time             `json:"created_at"`

	subs map[subscriber]struct{}
	mu   sync.RWMutex

	// Cancellation support.
	ctx    context.Context
	cancel context.CancelFunc

	// DB-backed state: used to persist progress
	db       *database.Queries
	courseID *int64
	moduleID *int64
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

func (j *GenerationJob) isCancelled() bool {
	select {
	case <-j.ctx.Done():
		return true
	default:
		return false
	}
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
	if req.Stage == "" {
		req.Stage = StageFull
	}
	job := &GenerationJob{
		ID:        JobID(uuid.New().String()),
		Status:    "pending",
		Stage:     req.Stage,
		Request:   req,
		CreatedAt: time.Now(),
		db:        s.db,
	}
	if req.ModuleID != 0 {
		mid := req.ModuleID
		job.moduleID = &mid
	}
	if req.CourseID != 0 {
		cid := req.CourseID
		job.courseID = &cid
	}
	// Persist to DB
	reqJSON, _ := json.Marshal(req)
	if err := s.db.CreateGenerationJob(context.Background(), string(job.ID), string(req.Stage), reqJSON, job.moduleID); err != nil {
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

// CancelJob cancels a running job by ID. Returns true if the job was found and cancelled.
func (s *JobStore) CancelJob(id JobID) bool {
	s.mu.RLock()
	job := s.inFlightMap[id]
	s.mu.RUnlock()
	if job == nil {
		return false
	}
	job.mu.Lock()
	if job.Status != "running" && job.Status != "pending" {
		job.mu.Unlock()
		return false
	}
	job.mu.Unlock()
	if job.cancel != nil {
		job.cancel()
	}
	return true
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
		Stage:  JobStage(row.Stage),
		db:     db,
	}
	if row.CourseID != nil {
		job.courseID = row.CourseID
	}
	if row.ModuleID != nil {
		job.moduleID = row.ModuleID
	}
	json.Unmarshal(row.Request, &job.Request)
	if job.Request.Stage == "" {
		job.Request.Stage = JobStage(row.Stage)
	}
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
	pool    *pgxpool.Pool
	store   *JobStore
	queries *database.Queries
	llm     *LLMClient
	emb     *embeddings.Client
	queue   chan *GenerationJob
}

func newWorker(pool *pgxpool.Pool, store *JobStore, queries *database.Queries, llm *LLMClient, emb *embeddings.Client) *Worker {
	w := &Worker{
		pool:    pool,
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

// completeModuleJob completes a StageModule job without the noisy audit/notify
// side effects of completeJob. Per-module regeneration would spam every user
// otherwise. The audit log for the original course creation already covers it.
func (w *Worker) completeModuleJob(job *GenerationJob, result gin.H, courseID, moduleID int64) {
	log.Printf("[worker] module job %s COMPLETED (module %d)", job.ID, moduleID)
	job.mu.Lock()
	job.Status = "completed"
	job.Result = result
	job.courseID = &courseID
	job.moduleID = &moduleID
	title := job.Request.Title
	job.mu.Unlock()
	resultJSON, _ := json.Marshal(result)
	w.queries.CompleteGenerationJob(context.Background(), string(job.ID), resultJSON, courseID)
	job.broadcast(SSEEvent{Event: "done", Data: result})

	audit.Log(w.queries, nil, "module_generated", map[string]any{
		"course_id": courseID,
		"module_id": moduleID,
		"title":     title,
	})
}

// runOutlinePipeline is the shared Step-1 flow used by the legacy one-shot
// (StageFull), the staged fresh outline (StageOutline with no CourseID),
// and the outline revision (StageOutline with CourseID + Feedback):
//
//	embed → broad retrieval → outliner LLM → persist course.
//
// Revision mode: when req.CourseID > 0 and req.Feedback is set, the current
// course is loaded, its modules + items are wiped, and the LLM is asked to
// revise the outline per the feedback. The existing course row is updated
// in place (preserving id, source_doc_ids, settings, created_by).
//
// Returns (courseID, coursePlan, sourceRefs, ok). On failure it calls
// failJob and ok=false; the caller must return immediately.
func (w *Worker) runOutlinePipeline(job *GenerationJob, req GenerateCourseRequest) (int64, plan, []gin.H, bool) {
	isRevision := req.CourseID > 0 && strings.TrimSpace(req.Feedback) != ""

	// For revisions we still embed the original description so retrieval
	// surfaces the same body of source material — the feedback is about
	// structure, not about pulling in different content.
	embedText := req.Description
	if isRevision {
		existing, err := w.queries.GetCourseByID(job.ctx, req.CourseID)
		if err != nil {
			log.Printf("[worker] revision: load course %d: %v", req.CourseID, err)
			w.failJob(job, "Course not found for revision")
			return 0, plan{}, nil, false
		}
		// Prefer the stored description (it may have been edited since the
		// original outline) for retrieval grounding.
		if strings.TrimSpace(existing.Description) != "" {
			embedText = existing.Description
		}
		if req.Title == "" || req.Title == "Untitled Course" {
			req.Title = existing.Title
		}
	}

	// --- Embed the course description (used for broad outline retrieval) ---
	if job.isCancelled() {
		w.failJob(job, "Cancelled")
		return 0, plan{}, nil, false
	}
	stepLabel := "Analyzing your course description..."
	if isRevision {
		stepLabel = "Re-analyzing source material for revision..."
	}
	job.addStep("embedding", stepLabel)
	embeddings, err := w.emb.Embed([]string{embedText})
	if err != nil {
		log.Printf("[worker] embed: %v", err)
		w.failJob(job, "Failed to analyze description")
		return 0, plan{}, nil, false
	}
	promptVec := pgvector.NewVector(float64ToFloat32(embeddings[0]))

	// Search only author-selected documents; no selection retains the existing
	// approved-corpus behavior.
	job.addStep("searching", "Searching approved documents for relevant content...")
	var chunks []database.SearchDocumentChunksRow
	if len(req.SourceDocIDs) > 0 {
		chunks, err = w.queries.SearchDocumentChunksByIDs(job.ctx, promptVec, req.SourceDocIDs, outlineChunkLimit)
	} else {
		chunks, err = w.queries.SearchDocumentChunks(job.ctx, database.SearchDocumentChunksParams{
			Embedding: promptVec,
			Limit:     outlineChunkLimit,
		})
	}
	if err != nil {
		log.Printf("[worker] search: %v", err)
		w.failJob(job, "Failed to search documents")
		return 0, plan{}, nil, false
	}
	if len(chunks) == 0 {
		job.addStep("searching", "No approved documents found. Using general knowledge...")
	}

	sourceContext := buildChunkContext(chunks)

	// =====================================================================
	// STEP 1: COURSE OUTLINER  (or revision)
	// =====================================================================
	if job.isCancelled() {
		w.failJob(job, "Cancelled")
		return 0, plan{}, nil, false
	}
	if isRevision {
		job.addStep("planning", "Revising course outline based on your feedback...")
	} else {
		job.addStep("planning", "Step 1/3: Creating course outline and module structure...")
	}
	log.Printf("[worker] job %s STEP 1 (%s): %s", job.ID, map[bool]string{true: "revision", false: "fresh"}[isRevision], req.Title)

	sourceCharCount := len(sourceContext)
	chunkCount := len(chunks)

	var planPrompt string
	var promptTemplate string
	if isRevision {
		promptTemplate = coursePlanRevisionPrompt
		// Build a compact rendering of the current outline so the LLM can see
		// what it's revising rather than starting from scratch.
		currentOutline := w.renderCurrentOutline(job.ctx, req.CourseID)
		targetCountInstruction := "No exact module count was requested; use the source-material scale."
		if targetCount, ok := requestedModuleCount(req.Feedback); ok {
			targetCountInstruction = fmt.Sprintf("HARD REQUIREMENT: the revised outline must contain exactly %d total modules. Treat this as a target total, not a number to add or remove.", targetCount)
		}
		planPrompt = fmt.Sprintf(`Current course title: %s
Current course description: %s

CURRENT OUTLINE:
%s

AUTHOR FEEDBACK:
%s

MODULE COUNT REQUIREMENT:
%s

Source material statistics: %d total characters across %d chunks.

Source material:
%s

Revise the outline per the author feedback. Output the full revised outline in the Markdown format.`,
			req.Title, embedText, currentOutline, strings.TrimSpace(req.Feedback), targetCountInstruction,
			sourceCharCount, chunkCount, sourceContext)
	} else {
		promptTemplate = coursePlanSystemPrompt
		planPrompt = fmt.Sprintf(`Course topic: %s
Course description/idea: %s

Source material statistics: %d total characters across %d chunks.

Source material:
%s

Generate a Markdown course plan PROPORTIONAL to the source material size shown above.`,
			req.Title, req.Description, sourceCharCount, chunkCount, sourceContext)
	}

	planResp, err := w.llm.Chat(promptTemplate, planPrompt)
	if err != nil {
		log.Printf("[worker] step1 outliner: %v", err)
		w.failJob(job, "Step 1 failed: outline generation error")
		return 0, plan{}, nil, false
	}

	coursePlan := parsePlanMarkdown(planResp)
	if len(coursePlan.Modules) == 0 {
		w.failJob(job, "Step 1 failed: no modules found")
		return 0, plan{}, nil, false
	}

	if targetCount, ok := requestedModuleCount(req.Feedback); ok && len(coursePlan.Modules) != targetCount {
		log.Printf("[worker] job %s step1: model returned %d modules, requested exactly %d; retrying", job.ID, len(coursePlan.Modules), targetCount)
		correctionPrompt := planPrompt + fmt.Sprintf(`

The previous response contained %d modules. It did not satisfy the hard requirement. Rewrite the FULL outline now with exactly %d total module headings. Do not add or remove that number from the existing outline; the final output itself must have %d modules.`, len(coursePlan.Modules), targetCount, targetCount)
		planResp, err = w.llm.Chat(promptTemplate, correctionPrompt)
		if err != nil {
			w.failJob(job, "Step 1 failed: outline revision error")
			return 0, plan{}, nil, false
		}
		coursePlan = parsePlanMarkdown(planResp)
		if len(coursePlan.Modules) != targetCount {
			w.failJob(job, fmt.Sprintf("Step 1 failed: outline must contain exactly %d modules", targetCount))
			return 0, plan{}, nil, false
		}
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
	if req.MaxQuestions > 0 {
		settingsMap["max_questions"] = req.MaxQuestions
	}
	settingsJSON, _ := json.Marshal(settingsMap)

	// --- Persist the course row (create or update) ---
	if isRevision {
		return req.CourseID, coursePlan, sourceRefs, true
	}

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
		return 0, plan{}, nil, false
	}

	return course.ID, coursePlan, sourceRefs, true
}

// renderCurrentOutline produces a compact Markdown rendering of the existing
// course outline so the revision LLM can see what it's editing. Items are
// omitted — we only show module structure since the revision target is the
// outline, not the generated content.
func (w *Worker) renderCurrentOutline(ctx context.Context, courseID int64) string {
	course, err := w.queries.GetCourseByID(ctx, courseID)
	if err != nil {
		return "(could not load existing outline)"
	}
	mods, err := w.queries.GetModulesByCourseWithStatus(ctx, courseID)
	if err != nil {
		mods = nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n%s\n", course.Title, course.Description)
	for i, m := range mods {
		fmt.Fprintf(&b, "\n## Module %d: %s\n**Description:** %s\n", i+1, m.Title, m.Description)
	}
	if len(mods) == 0 {
		b.WriteString("\n(no modules yet)\n")
	}
	return b.String()
}

// processJob is the worker entrypoint. It dispatches to one of three
// pipelines based on the job's Stage:
//   - StageFull:    legacy one-shot outline + all modules (backwards compat)
//   - StageOutline: outline + module rows only; stops before content gen
//   - StageModule:  generate content/items for one existing module row
//
// All stages share the same SSE/DB lifecycle (StartGenerationJob, complete/fail).
func (w *Worker) processJob(job *GenerationJob) {
	job.mu.Lock()
	job.Status = "running"
	job.ctx, job.cancel = context.WithCancel(context.Background())
	reqTitle := job.Request.Title
	stage := job.Stage
	job.mu.Unlock()
	defer job.cancel() // clean up context
	w.queries.StartGenerationJob(context.Background(), string(job.ID))

	// Notify admins that generation has started (skipped for module-stage
	// jobs: those are small, fast, and would spam notifications otherwise).
	if stage != StageModule {
		notifyAdmins(w.queries,
			"Course generation started",
			fmt.Sprintf("AI is generating \"%s\".", reqTitle),
			"/ai-content-generator",
		)
	}

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

	switch stage {
	case StageModule:
		w.runModuleJob(job, req)
	case StageOutline:
		w.runOutlineJob(job, req)
	case StageFull:
		fallthrough
	default:
		w.runFullJob(job, req)
	}
}

// runFullJob is the legacy one-shot pipeline: outline, create course, then
// generate every module in parallel. Kept for backwards compatibility and as
// a "quick generate" mode for users who want the old behaviour.
func (w *Worker) runFullJob(job *GenerationJob, req GenerateCourseRequest) {
	courseID, coursePlan, sourceRefs, ok := w.runOutlinePipeline(job, req)
	if !ok {
		return // failure already reported via failJob
	}

	if job.isCancelled() {
		w.failJob(job, "Cancelled")
		return
	}
	if req.CourseID != 0 && strings.TrimSpace(req.Feedback) != "" {
		modules, ok := w.replaceCourseOutline(job, req, courseID, coursePlan, sourceRefs)
		if !ok {
			return
		}
		w.completeJob(job, gin.H{
			"id":             courseID,
			"title":          coursePlan.Title,
			"description":    coursePlan.Description,
			"status":         "draft",
			"source_doc_ids": req.SourceDocIDs,
			"sources":        sourceRefs,
			"modules":        modules,
			"stage":          "outline",
		}, courseID)
		return
	}

	// STEP 2/3: Generate each module IN PARALLEL.
	// Each module retrieves its OWN source chunks (per-module retrieval), so
	// a large document is actually covered instead of every module rewriting
	// the same handful of chunks. modulesResult is indexed by module position
	// so final ordering is preserved regardless of completion order.
	totalModules := len(coursePlan.Modules)
	modulesResult := make([]gin.H, totalModules)

	g, gctx := errgroup.WithContext(job.ctx)
	g.SetLimit(moduleConcurrency)

	for mi, modPlan := range coursePlan.Modules {
		mi, modPlan := mi, modPlan
		g.Go(func() error {
			maxConcepts := moduleQuestionBudget(req.MaxQuestions, mi, totalModules)
			modResult, err := w.generateModule(gctx, job, courseID, coursePlan, req.SourceDocIDs, mi, modPlan, totalModules, req.QuestionTypes, nil, maxConcepts, nil)
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
		"source_doc_ids": req.SourceDocIDs,
		"sources":        sourceRefs,
		"modules":        modulesResult,
	}
	w.completeJob(job, result, courseID)
}

func (w *Worker) replaceCourseOutline(job *GenerationJob, req GenerateCourseRequest, courseID int64, coursePlan plan, sourceRefs []gin.H) ([]gin.H, bool) {
	job.addStep("saving", fmt.Sprintf("Replacing outline: %s (%d modules)...", coursePlan.Title, len(coursePlan.Modules)))
	tx, err := w.pool.Begin(job.ctx)
	if err != nil {
		log.Printf("[worker] revision: begin transaction: %v", err)
		w.failJob(job, "Failed to prepare course revision")
		return nil, false
	}
	defer tx.Rollback(context.Background())
	queries := w.queries.WithTx(tx)

	settings := map[string]interface{}{"sources": sourceRefs}
	if req.MaxQuestions > 0 {
		settings["max_questions"] = req.MaxQuestions
	}
	settingsJSON, _ := json.Marshal(settings)
	if err := queries.DeleteCourseItems(job.ctx, courseID); err != nil {
		log.Printf("[worker] revision: delete items: %v", err)
		w.failJob(job, "Failed to replace course outline")
		return nil, false
	}
	if err := queries.DeleteCourseModules(job.ctx, courseID); err != nil {
		log.Printf("[worker] revision: delete modules: %v", err)
		w.failJob(job, "Failed to replace course outline")
		return nil, false
	}
	if _, err := queries.UpdateCourseSettings(job.ctx, database.UpdateCourseSettingsParams{ID: courseID, Settings: settingsJSON}); err != nil {
		log.Printf("[worker] revision: update settings: %v", err)
		w.failJob(job, "Failed to replace course outline")
		return nil, false
	}
	if _, err := queries.UpdateCourseMeta(job.ctx, database.UpdateCourseMetaParams{ID: courseID, Title: coursePlan.Title, Description: coursePlan.Description}); err != nil {
		log.Printf("[worker] revision: update metadata: %v", err)
		w.failJob(job, "Failed to replace course outline")
		return nil, false
	}

	modules := make([]gin.H, 0, len(coursePlan.Modules))
	for index, modulePlan := range coursePlan.Modules {
		if job.isCancelled() {
			w.failJob(job, "Cancelled")
			return nil, false
		}
		module, err := queries.CreateModule(job.ctx, database.CreateModuleParams{
			CourseID: courseID, Title: modulePlan.Title, Description: modulePlan.Description, SortOrder: int32(index),
		})
		if err != nil {
			log.Printf("[worker] revision: create module %d: %v", index+1, err)
			w.failJob(job, "Failed to replace course outline")
			return nil, false
		}
		modules = append(modules, gin.H{"id": module.ID, "title": modulePlan.Title, "description": modulePlan.Description, "sort_order": int32(index), "status": "pending", "items": []gin.H{}})
	}
	if err := tx.Commit(job.ctx); err != nil {
		log.Printf("[worker] revision: commit transaction: %v", err)
		w.failJob(job, "Failed to replace course outline")
		return nil, false
	}
	return modules, true
}

// runOutlineJob runs the outline pipeline and then creates one modules row
// per planned module (status='pending') so the user can review/edit before
// any content is generated.
//
// Two modes, both driven by the request:
//   - Fresh outline: req.CourseID == 0. A new course row is created.
//   - Revision:      req.CourseID > 0 && req.Feedback != "". The existing
//     course's modules + items are wiped and replaced with the revised
//     outline. The course row itself is preserved (same id, settings, etc.).
//
// The job completes with a result describing the outline; the frontend can
// then drive per-module generation by enqueuing StageModule jobs.
func (w *Worker) runOutlineJob(job *GenerationJob, req GenerateCourseRequest) {
	courseID, coursePlan, sourceRefs, ok := w.runOutlinePipeline(job, req)
	if !ok {
		return
	}

	if job.isCancelled() {
		w.failJob(job, "Cancelled")
		return
	}

	// Create module rows with status='pending' so the staged builder UI can
	// show them awaiting generation. We don't write any content/items here.
	job.addStep("saving", fmt.Sprintf("Saving %d module outlines...", len(coursePlan.Modules)))
	modules := make([]gin.H, 0, len(coursePlan.Modules))
	for mi, modPlan := range coursePlan.Modules {
		if job.isCancelled() {
			w.failJob(job, "Cancelled")
			return
		}
		module, err := w.queries.CreateModule(job.ctx, database.CreateModuleParams{
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
		// Mark as pending outline row (default is already 'pending' but be explicit)
		if err := w.queries.UpdateModuleStatus(job.ctx, module.ID, "pending"); err != nil {
			log.Printf("[worker] set module %d status: %v", module.ID, err)
		}
		modules = append(modules, gin.H{
			"id":          module.ID,
			"title":       modPlan.Title,
			"description": modPlan.Description,
			"sort_order":  int32(mi),
			"status":      "pending",
			"items":       []gin.H{},
		})
	}

	result := gin.H{
		"id":             courseID,
		"title":          coursePlan.Title,
		"description":    coursePlan.Description,
		"status":         "draft",
		"source_doc_ids": req.SourceDocIDs,
		"sources":        sourceRefs,
		"modules":        modules,
		"stage":          "outline",
	}
	w.completeJob(job, result, courseID)
}

// runModuleJob generates content + items for ONE existing module row.
// The module must already exist (created by an outline-stage job). Items
// attached to the module are deleted first so regeneration is clean.
func (w *Worker) runModuleJob(job *GenerationJob, req GenerateCourseRequest) {
	if req.CourseID == 0 || req.ModuleID == 0 {
		w.failJob(job, "course_id and module_id are required for module-stage jobs")
		return
	}

	module, err := w.queries.GetModule(job.ctx, req.ModuleID)
	if err != nil {
		log.Printf("[worker] module job: load module %d: %v", req.ModuleID, err)
		w.failJob(job, "Module not found")
		return
	}
	if module.CourseID != req.CourseID {
		w.failJob(job, "module does not belong to this course")
		return
	}

	// Look up the course so we can build the same prompts the full pipeline uses.
	course, err := w.queries.GetCourseByID(job.ctx, req.CourseID)
	if err != nil {
		log.Printf("[worker] module job: load course %d: %v", req.CourseID, err)
		w.failJob(job, "Course not found")
		return
	}

	// Claim the module as generating. If another job is already running for
	// this module, bail out to avoid duplicate work.
	if module.Status == "generating" {
		w.failJob(job, "this module is already being generated")
		return
	}
	if err := w.queries.UpdateModuleStatus(job.ctx, module.ID, "generating"); err != nil {
		log.Printf("[worker] module job: set status generating: %v", err)
	}

	// Wipe any existing items so regeneration produces a clean module.
	if err := w.queries.DeleteCourseItemsByModule(job.ctx, pgtype.Int8{Int64: module.ID, Valid: true}); err != nil {
		log.Printf("[worker] module job: clear old items: %v", err)
	}

	coursePlan := plan{Title: course.Title, Description: course.Description}
	modPlan := planModule{Title: module.Title, Description: module.Description}

	// Module-row question types take precedence over the request body. This
	// lets the user configure types once in the outline review and have them
	// stick across regenerations, instead of having to re-pass them each time.
	questionTypes := req.QuestionTypes
	if len(module.QuestionTypes) > 0 {
		questionTypes = module.QuestionTypes
	}

	maxConcepts := 0
	if total := maxQuestionsFromSettings(course.Settings); total > 0 {
		mods, _ := w.queries.GetModulesByCourseWithStatus(job.ctx, req.CourseID)
		n := len(mods)
		if n < 1 {
			n = 1
		}
		maxConcepts = moduleQuestionBudget(total, int(module.SortOrder), n)
		items, err := w.queries.GetCourseItemsByCourseWithGroup(job.ctx, req.CourseID)
		if err != nil {
			log.Printf("[worker] module job: count existing concepts: %v", err)
			w.failJob(job, "Failed to enforce course question limit")
			return
		}
		remaining := total - generatedConceptCount(items)
		if remaining < maxConcepts {
			maxConcepts = max(remaining, 0)
		}
	}

	modResult, err := w.generateModule(job.ctx, job, req.CourseID, coursePlan, course.SourceDocIds, int(module.SortOrder), modPlan, 1, questionTypes, module.QuestionTypeTargets, maxConcepts, module)
	if err != nil {
		// Mark the module as failed but report the error on the job too.
		_ = w.queries.UpdateModuleStatus(context.Background(), module.ID, "failed")
		w.failJob(job, err.Error())
		return
	}

	if err := w.queries.UpdateModuleStatus(context.Background(), module.ID, "ready"); err != nil {
		log.Printf("[worker] module job: set status ready: %v", err)
	}

	result := gin.H{
		"id":     req.CourseID,
		"module": modResult,
		"stage":  "module",
	}
	w.completeModuleJob(job, result, req.CourseID, module.ID)
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
// for a single module. Each section gets its content written, then per-concept
// question-type variants are generated for that section. Items are stored
// interleaved: content, q1 variants, content, q2 variants...
//
// maxConcepts is a HARD cap on the number of unique questions (concepts) this
// module may produce. 0 = no cap. It's the per-module share of the course's
// optional max_questions budget, enforced so the author gets a predictable
// total question count regardless of how many sections the LLM plans.
//
// If `existing` is non-nil, the module row is reused (StageModule path); otherwise a
// new modules row is created (legacy StageFull path).
func (w *Worker) generateModule(ctx context.Context, job *GenerationJob, courseID int64, coursePlan plan, sourceDocIDs []int64, mi int, modPlan planModule, totalModules int, questionTypes []string, questionTypeTargets map[string]int, maxConcepts int, existing *database.ModuleWithStatus) (gin.H, error) {
	moduleLabel := fmt.Sprintf("%d/%d", mi+1, totalModules)

	// --- Per-module retrieval ---
	job.addStep("retrieving", fmt.Sprintf("Finding source material for Module %s — %s...", moduleLabel, modPlan.Title))
	modEmb, err := w.emb.Embed([]string{modPlan.Title + ". " + modPlan.Description})
	if err != nil {
		log.Printf("[worker] module %s embed: %v", moduleLabel, err)
		return nil, fmt.Errorf("Failed to retrieve material for module %d", mi+1)
	}
	modVec := pgvector.NewVector(float64ToFloat32(modEmb[0]))

	var modChunks []database.SearchDocumentChunksRow
	if len(sourceDocIDs) > 0 {
		modChunks, err = w.queries.SearchDocumentChunksByIDs(ctx, modVec, sourceDocIDs, moduleChunkLimit)
	} else {
		modChunks, err = w.queries.SearchDocumentChunks(ctx, database.SearchDocumentChunksParams{
			Embedding: modVec,
			Limit:     moduleChunkLimit,
		})
	}
	if err != nil {
		log.Printf("[worker] module %s search: %v", moduleLabel, err)
		return nil, fmt.Errorf("Failed to retrieve material for module %d", mi+1)
	}
	moduleContext := buildChunkContext(modChunks)

	var moduleID int64
	var moduleSort int32
	if existing != nil {
		// StageModule path: reuse the existing row created by the outline job.
		moduleID = existing.ID
		moduleSort = existing.SortOrder
	} else {
		// StageFull path: create the modules row inline (legacy behaviour).
		created, err := w.queries.CreateModule(ctx, database.CreateModuleParams{
			CourseID:    courseID,
			Title:       modPlan.Title,
			Description: modPlan.Description,
			SortOrder:   int32(mi),
		})
		if err != nil {
			log.Printf("[worker] create module %d: %v", mi+1, err)
			return nil, fmt.Errorf("Failed to create module %d", mi+1)
		}
		moduleID = created.ID
		moduleSort = created.SortOrder
	}

	// --- Step 2a: Section Lister ---
	if job.isCancelled() {
		return nil, fmt.Errorf("Cancelled")
	}
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

	// Build allowed question types from the request (empty = allow all).
	allKnownTypes := map[string]string{
		"mc": "multiple choice", "ma": "multiple answer", "tf": "true/false",
		"fb": "fill-in-the-blank", "sa": "short answer", "matching": "matching",
		"drag_sort": "drag and sort", "hotspot": "hotspot",
	}
	allowedTypes := make(map[string]bool)
	if len(questionTypes) == 0 {
		for k := range allKnownTypes {
			allowedTypes[k] = true
		}
	} else {
		for _, qt := range questionTypes {
			qt = strings.ToLower(strings.TrimSpace(qt))
			if _, ok := allKnownTypes[qt]; ok {
				allowedTypes[qt] = true
			}
		}
	}
	typeTargets := make(map[string]int)
	for questionType, target := range questionTypeTargets {
		questionType = strings.ToLower(strings.TrimSpace(questionType))
		if allowedTypes[questionType] && target > 0 {
			typeTargets[questionType] = target
		}
	}
	typeCounts := make(map[string]int)

	// Accumulate AI-reported skips across all sections of this module.
	// Persisted to modules.limitations at the end so the UI can surface them
	// as warnings (e.g. "matching skipped on section 2: no natural pairs").
	limitations := make([]gin.H, 0)

	// Hard question cap: count unique concepts created so we can stop once the
	// module's share of the course budget is reached. Content for later
	// sections still gets written (it's teaching material), but no further
	// question concepts are emitted past the cap.
	conceptsCreated := 0

	for si, secTitle := range sectionTitles {
		// Check for cancellation before each section.
		if job.isCancelled() {
			return nil, fmt.Errorf("Cancelled")
		}
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
			ModuleID:  pgtype.Int8{Int64: moduleID, Valid: true},
			ItemType:  "content",
			SortOrder: sortOrder,
			Data:      []byte(contentData),
		})
		if err == nil {
			contentItem := gin.H{
				"id": ciContent.ID, "item_type": "content", "sort_order": sortOrder,
				"data": json.RawMessage(contentData),
			}
			itemsResult = append(itemsResult, contentItem)
			// Live preview: broadcast the new item so the UI can render it as
			// soon as it's persisted, not just when the job finishes.
			job.broadcast(SSEEvent{Event: "item", Data: gin.H{
				"module_id": moduleID,
				"section":   secTitle,
				"item":      contentItem,
			}})
			sortOrder++
		} else {
			log.Printf("[worker] create content item (module %d section %d): %v", mi+1, si+1, err)
		}

		// --- Generate per-concept question-type variants ---
		// Adaptive learning: each testable concept in the section is rendered in
		// EVERY question type that fits it (variants share a question_group_id),
		// so the engine can later serve a learner their preferred/best type for
		// that concept instead of always the same format.

		// Enforce the hard question cap. Content for this section was already
		// written; if the module's concept budget is exhausted, skip question
		// generation for this and later sections so the course stays within the
		// author's requested limit.
		if maxConcepts > 0 && conceptsCreated >= maxConcepts {
			log.Printf("[worker] module %s: question cap %d reached — skipping questions for section %d", moduleLabel, maxConcepts, si+1)
			continue
		}

		job.addStep("writing", fmt.Sprintf("Creating question variants for section %d/%d of Module %s...",
			si+1, len(sectionTitles), moduleLabel))

		contentSummary := sectionBody
		if len(contentSummary) > 2000 {
			contentSummary = contentSummary[:2000]
		}

		// Build type restriction for the prompt (when the module narrows types).
		typeList := make([]string, 0, len(allowedTypes))
		for k := range allowedTypes {
			typeList = append(typeList, fmt.Sprintf("%s (%s)", k, allKnownTypes[k]))
		}
		var typeRestriction string
		if len(typeList) < len(allKnownTypes) {
			typeRestriction = fmt.Sprintf("\n\nCRITICAL: ONLY use these question types: %s. Do NOT use any other types.", strings.Join(typeList, ", "))
		}

		// When a hard cap is in effect, tell the model exactly how many more
		// concepts it may emit for this module so it doesn't over-produce (the
		// loop below also hard-truncates as a safety net).
		var capRestriction string
		if maxConcepts > 0 {
			remaining := maxConcepts - conceptsCreated
			capRestriction = fmt.Sprintf("\n\nCRITICAL: Produce AT MOST %d distinct concept(s) for this section. Fewer is fine; never exceed this number.", remaining)
		}
		var targetRestriction string
		if len(typeTargets) > 0 {
			remainingTargets := make([]string, 0, len(typeTargets))
			for questionType, target := range typeTargets {
				if remaining := target - typeCounts[questionType]; remaining > 0 {
					remainingTargets = append(remainingTargets, fmt.Sprintf("%s: %d", questionType, remaining))
				}
			}
			if len(remainingTargets) > 0 {
				targetRestriction = fmt.Sprintf("\n\nQUESTION-TYPE TARGETS FOR THE REMAINDER OF THIS MODULE: %s. Produce as many valid variants as possible toward these targets, without exceeding any target.", strings.Join(remainingTargets, ", "))
			}
		}

		questionPrompt := fmt.Sprintf(`Section: "%s"

Content:
%s

Identify the distinct concepts in this section and render each one in every question type that fits it.%s%s%s`, secTitle, contentSummary, typeRestriction, capRestriction, targetRestriction)

		questionResp, err := w.llm.Chat(conceptVariantPrompt, questionPrompt)
		if err != nil {
			log.Printf("[worker] questions (m%d s%d): %v", mi+1, si+1, err)
			// Non-fatal: continue to next section even if questions fail
			continue
		}

		questionJSON := stripMarkdownFences(questionResp)

		// Preferred shape: {"concepts": [...], "limitations": [...]}. Fall back to
		// the legacy {"items": [...]} (or bare array) shape if the model ignores
		// the concept structure — those become standalone items with no group.
		var concepts []conceptGroup
		var sectionLimits []gin.H
		var conceptWrapper struct {
			Concepts    []conceptGroup `json:"concepts"`
			Limitations []gin.H        `json:"limitations"`
		}
		var legacyItems []genItem
		var legacyWrapper struct {
			Items       []genItem `json:"items"`
			Limitations []gin.H   `json:"limitations"`
		}
		parsedConcepts := false
		if err := json.Unmarshal([]byte(questionJSON), &conceptWrapper); err == nil && len(conceptWrapper.Concepts) > 0 {
			concepts = conceptWrapper.Concepts
			sectionLimits = conceptWrapper.Limitations
			parsedConcepts = true
		} else if err := json.Unmarshal([]byte(questionJSON), &legacyWrapper); err == nil && legacyWrapper.Items != nil {
			legacyItems = legacyWrapper.Items
			sectionLimits = legacyWrapper.Limitations
		} else if err := json.Unmarshal([]byte(questionJSON), &legacyItems); err != nil {
			log.Printf("[worker] parse questions (m%d s%d): %v — skipping questions", mi+1, si+1, err)
			continue
		}

		// Tag each limitation with the section it came from so the user can
		// see WHERE the AI balked, not just that it did.
		for _, l := range sectionLimits {
			if _, ok := l["section"]; !ok {
				l["section"] = secTitle
			}
			limitations = append(limitations, l)
		}

		// Helper to persist one variant item and stream it to the live preview.
		createVariant := func(q genItem, groupID string) {
			qType := strings.ToLower(strings.TrimSpace(q.Type))
			if !allowedTypes[qType] {
				log.Printf("[worker] disallowed item_type %q (m%d s%d) — skipping", q.Type, mi+1, si+1)
				limitations = append(limitations, gin.H{"type": q.Type, "reason": "The generated type was outside the module's allowed question types.", "section": secTitle})
				return
			}
			if target, limited := typeTargets[qType]; limited && typeCounts[qType] >= target {
				return
			}
			itemData := ensureIRTParams(q.Data, qType)
			ci, err := w.queries.CreateCourseItemWithGroup(ctx, database.CreateCourseItemWithGroupParams{
				CourseID:        courseID,
				ModuleID:        pgtype.Int8{Int64: moduleID, Valid: true},
				ItemType:        qType,
				SortOrder:       sortOrder,
				Data:            itemData,
				QuestionGroupID: groupID,
			})
			if err != nil {
				log.Printf("[worker] create question (m%d s%d): %v", mi+1, si+1, err)
				return
			}
			questionItem := gin.H{
				"id": ci.ID, "item_type": qType, "sort_order": sortOrder,
				"question_group_id": groupID,
				"data":              json.RawMessage(itemData),
			}
			itemsResult = append(itemsResult, questionItem)
			typeCounts[qType]++
			job.broadcast(SSEEvent{Event: "item", Data: gin.H{
				"module_id": moduleID,
				"section":   secTitle,
				"item":      questionItem,
			}})
			sortOrder++
		}

		if parsedConcepts {
			for ciIdx, c := range concepts {
				if len(c.Variants) == 0 {
					continue
				}
				// Hard cap safety net: stop once the module's concept budget is
				// reached, even if the model ignored the per-section instruction.
				if maxConcepts > 0 && conceptsCreated >= maxConcepts {
					log.Printf("[worker] module %s: truncating concepts at cap %d", moduleLabel, maxConcepts)
					break
				}
				// Stable group id scoped to course/module/section/concept so
				// regeneration produces clean, non-colliding groups.
				conceptID := strings.TrimSpace(c.ConceptID)
				if conceptID == "" {
					conceptID = fmt.Sprintf("c%d", ciIdx+1)
				}
				groupID := fmt.Sprintf("%d-m%d-s%d-%s", courseID, moduleID, si, conceptID)
				for _, v := range c.Variants {
					createVariant(v, groupID)
				}
				conceptsCreated++
			}
		} else {
			// Legacy fallback: standalone items, not grouped.
			for _, q := range legacyItems {
				if maxConcepts > 0 && conceptsCreated >= maxConcepts {
					break
				}
				createVariant(q, "")
				conceptsCreated++
			}
		}
	}
	for questionType, target := range typeTargets {
		if generated := typeCounts[questionType]; generated < target {
			limitations = append(limitations, gin.H{
				"type":   questionType,
				"reason": fmt.Sprintf("Generated %d of %d requested items from the available source material.", generated, target),
			})
		}
	}

	// Persist AI-reported limitations so the UI can show warnings like
	// "matching skipped on 2 sections". We also clear stale limitations from
	// any prior generation pass.
	if limitsJSON, err := json.Marshal(limitations); err == nil {
		if err := w.queries.UpdateModuleLimitations(ctx, moduleID, limitsJSON); err != nil {
			log.Printf("[worker] persist limitations (m%d): %v", mi+1, err)
		}
	}

	modResult := gin.H{
		"id":          moduleID,
		"title":       modPlan.Title,
		"description": modPlan.Description,
		"sort_order":  moduleSort,
		"items":       itemsResult,
		"limitations": limitations,
	}
	job.addModule(modResult, moduleLabel)
	return modResult, nil
}

func float64ToFloat32Job(in []float64) []float32 {
	return float64ToFloat32(in)
}
