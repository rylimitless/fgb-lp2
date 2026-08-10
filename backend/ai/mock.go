package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	database "fgb-lp/database/queries"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

// MockSizePreset is a fixed course shape. Using presets (instead of fully free
// form) keeps runs comparable: "small" today costs the same shape as "small"
// next week, so the spend dashboard can meaningfully average across them.
type MockSizePreset struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	Description    string `json:"description"`
	Modules        int    `json:"modules"`
	ItemsPerMod    int    `json:"items_per_module"`
	SectionsPerMod int    `json:"sections_per_module"`
}

// MockPresets is the canonical menu the UI renders. The frontend mirrors this
// list; the backend is the source of truth.
var MockPresets = []MockSizePreset{
	{
		ID:          "tiny",
		Label:       "Tiny",
		Description: "1 module · 1 section · 2 items. Cheapest sanity check (~3 LLM calls).",
		Modules:     1, SectionsPerMod: 1, ItemsPerMod: 2,
	},
	{
		ID:          "small",
		Label:       "Small",
		Description: "2 modules · 2 sections each · 3 items per module. Realistic smallest course.",
		Modules:     2, SectionsPerMod: 2, ItemsPerMod: 3,
	},
	{
		ID:          "medium",
		Label:       "Medium",
		Description: "4 modules · 3 sections each · 5 items per module. Typical production course.",
		Modules:     4, SectionsPerMod: 3, ItemsPerMod: 5,
	},
	{
		ID:          "large",
		Label:       "Large",
		Description: "6 modules · 4 sections each · 8 items per module. Stress-test pricing.",
		Modules:     6, SectionsPerMod: 4, ItemsPerMod: 8,
	},
}

// LookupPreset returns the named preset, or nil if not found.
func LookupPreset(id string) *MockSizePreset {
	for i := range MockPresets {
		if MockPresets[i].ID == id {
			return &MockPresets[i]
		}
	}
	return nil
}

// MockRunStats is the aggregate token accounting for one mock-course run. This
// is what the UI uses to compute spend (tokens × per-million price) and to
// show "average per course" across historical runs.
type MockRunStats struct {
	JobID            string  `json:"job_id"`
	PresetID         string  `json:"preset_id"`
	CourseID         int64   `json:"course_id"`
	Model            string  `json:"model"`
	LLMCalls         int     `json:"llm_calls"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
	CachedPrompt     int64   `json:"cached_prompt_tokens"`
	ReasoningTokens  int64   `json:"reasoning_tokens"`
	DurationSeconds  float64 `json:"duration_seconds"`
	ModulesCreated   int     `json:"modules_created"`
	ItemsCreated     int     `json:"items_created"`
}

// tokenRecorder is the shared sink every LLM call in a mock run reports to. It
// both persists each call to llm_token_usage (so historical aggregation works)
// and accumulates the in-memory totals returned at the end of the run.
type tokenRecorder struct {
	mu       sync.Mutex
	queries  *database.Queries
	model    string
	source   string
	jobRef   string
	courseID *int64
	userID   *int64
	stats    MockRunStats
}

func newTokenRecorder(q *database.Queries, model, jobRef string, courseID, userID *int64) *tokenRecorder {
	return &tokenRecorder{
		queries:  q,
		model:    model,
		source:   "mock_course",
		jobRef:   jobRef,
		courseID: courseID,
		userID:   userID,
		stats: MockRunStats{
			Model:    model,
			JobID:    jobRef,
			LLMCalls: 0,
		},
	}
}

// record logs one LLM call's usage to the DB and folds it into the running
// totals. Label is a short human-readable hint shown in the live log
// ("outline", "module-2:section-1:content", ...).
func (r *tokenRecorder) record(ctx context.Context, label string, u Usage) {
	r.mu.Lock()
	r.stats.LLMCalls++
	r.stats.PromptTokens += u.PromptTokens
	r.stats.CompletionTokens += u.CompletionTokens
	r.stats.TotalTokens += u.TotalTokens
	r.stats.CachedPrompt += u.CachedPromptTokens
	r.stats.ReasoningTokens += u.ReasoningTokens
	r.mu.Unlock()

	if _, err := r.queries.LogLLMTokenUsage(ctx, database.LogLLMTokenUsageParams{
		Model:              r.model,
		Source:             r.source,
		Label:              label,
		PromptTokens:       u.PromptTokens,
		CompletionTokens:   u.CompletionTokens,
		TotalTokens:        u.TotalTokens,
		CachedPromptTokens: u.CachedPromptTokens,
		ReasoningTokens:    u.ReasoningTokens,
		CourseID:           r.courseID,
		UserID:             r.userID,
		JobRef:             &r.jobRef,
	}); err != nil {
		// Logging is best-effort: don't fail the run if the DB hiccups, but
		// surface it so an admin notices the spend dashboard is incomplete.
		log.Printf("[mock_course] token-usage log failed: %v", err)
	}
}

// mockOutlineSystem is the system prompt for the outline step. Deliberately
// shorter than the real coursePlanSystemPrompt — the goal is realistic token
// accounting at a controlled size, not production-quality output.
const mockOutlineSystem = `You are a course outline generator. Return ONLY valid minified JSON, no markdown fences. Shape: {"title": string, "description": string, "modules": [{"title": string, "description": string}]}. Each description is one sentence.`

// mockContentSystem generates one section of body content.
const mockContentSystem = `You are a course content writer. Return ONLY the body text of one section, plain prose, no headings, 60-120 words.`

// mockQuestionSystem generates one multiple-choice question as JSON.
const mockQuestionSystem = `You are an assessment writer. Return ONLY valid minified JSON, no markdown fences. Shape: {"stem": string, "options": [string, string, string, string], "answerIndex": 0, "explanation": string}. The stem tests the concept in the section. answerIndex is 0-based.`

// topicBank gives the mock generator varied subject matter so cache-hit
// pricing (which kicks in on repeated system prompts + repeated prefixes)
// behaves realistically instead of always hitting the same content.
var topicBank = []string{
	"Introduction to Cloud Computing",
	"Foundations of Data Analysis with SQL",
	"Workplace Cybersecurity Essentials",
	"Effective Business Communication",
	"Project Management Fundamentals",
	"Customer Service Excellence",
	"Financial Literacy for Non-Finance Managers",
	"Leadership and Team Dynamics",
	"Digital Marketing Basics",
	"Time Management and Productivity",
	"Negotiation Skills for Professionals",
	"Sustainable Business Practices",
}

// runMockCourse generates a course of the requested preset size using real LLM
// calls, persists it as a draft course, and streams progress to the SSE writer.
// Every LLM call's token usage is recorded via the tokenRecorder.
//
// This is intentionally NOT the production course pipeline — it uses simpler
// prompts and a fixed shape so runs are comparable to each other for spend
// benchmarking. The real pipeline lives in jobs.go.
func (h *Handler) runMockCourse(
	ctx context.Context,
	preset MockSizePreset,
	jobRef string,
	userID int64,
	sw *sseWriter,
) (MockRunStats, error) {
	start := time.Now()
	topic := topicBank[rand.Intn(len(topicBank))]

	sw.send("step", gin.H{"detail": fmt.Sprintf("Generating outline for %q (%s)", topic, preset.Label)})

	// --- 1. Outline -------------------------------------------------------
	outlinePrompt := fmt.Sprintf(
		`Create a course outline on "%s". Generate exactly %d modules. Each module description must be one sentence.`,
		topic, preset.Modules,
	)
	outlineResp, outlineUsage, err := h.LLM.ChatWithUsage(mockOutlineSystem, outlinePrompt)
	if err != nil {
		return MockRunStats{}, fmt.Errorf("outline LLM call: %w", err)
	}

	course, err := h.Queries.CreateCourse(ctx, database.CreateCourseParams{
		Title:        fmt.Sprintf("[MOCK · %s] %s", preset.Label, topic),
		Description:  fmt.Sprintf("Auto-generated mock course (preset: %s). Created by developer tools for LLM spend benchmarking.", preset.Label),
		CreatedBy:    userID,
		SourceDocIds: []int64{},
		Settings:     []byte(`{"mock": true, "preset": "` + preset.ID + `"}`),
	})
	if err != nil {
		return MockRunStats{}, fmt.Errorf("create course: %w", err)
	}
	courseID := course.ID
	recorder := newTokenRecorder(h.Queries, h.LLM.Model(), jobRef, &courseID, &userID)
	recorder.record(ctx, "outline", outlineUsage)
	recorder.stats.PresetID = preset.ID
	recorder.stats.CourseID = courseID
	sw.send("tokens", gin.H{
		"label":             "outline",
		"prompt_tokens":     outlineUsage.PromptTokens,
		"completion_tokens": outlineUsage.CompletionTokens,
		"total_tokens":      outlineUsage.TotalTokens,
		"running_total":     recorder.stats.TotalTokens,
	})

	// Parse the outline. If the model misbehaves, fall back to deterministic
	// module names so the run can still complete and produce token stats.
	modules := parseMockOutline(outlineResp, preset.Modules, topic)
	sw.send("outline", gin.H{
		"course_id":    courseID,
		"title":        course.Title,
		"module_count": len(modules),
	})

	// --- 2. Modules (sections + items) -----------------------------------
	for mi, mod := range modules {
		if ctx.Err() != nil {
			return recorder.stats, ctx.Err()
		}
		modLabel := fmt.Sprintf("module-%d", mi+1)
		sw.send("step", gin.H{"detail": fmt.Sprintf("Module %d/%d: %s", mi+1, preset.Modules, mod.Title)})

		dbMod, err := h.Queries.CreateModule(ctx, database.CreateModuleParams{
			CourseID:    courseID,
			Title:       mod.Title,
			Description: mod.Description,
			SortOrder:   int32(mi),
		})
		if err != nil {
			return recorder.stats, fmt.Errorf("create module %d: %w", mi+1, err)
		}
		recorder.stats.ModulesCreated++

		itemsCreated := 0
		itemsPerSection := preset.ItemsPerMod / preset.SectionsPerMod
		if itemsPerSection < 1 {
			itemsPerSection = 1
		}

		for si := 0; si < preset.SectionsPerMod; si++ {
			if ctx.Err() != nil {
				return recorder.stats, ctx.Err()
			}
			secLabel := fmt.Sprintf("%s:section-%d", modLabel, si+1)

			// 2a. Content for the section.
			contentPrompt := fmt.Sprintf(
				`Write a section titled "Section %d" for the module "%s" (course: "%s"). Cover one concrete subtopic.`,
				si+1, mod.Title, topic,
			)
			contentResp, contentUsage, err := h.LLM.ChatWithUsage(mockContentSystem, contentPrompt)
			if err != nil {
				return recorder.stats, fmt.Errorf("content LLM call (%s): %w", secLabel, err)
			}
			recorder.record(ctx, secLabel+":content", contentUsage)
			sw.send("tokens", gin.H{
				"label":             secLabel + ":content",
				"prompt_tokens":     contentUsage.PromptTokens,
				"completion_tokens": contentUsage.CompletionTokens,
				"total_tokens":      contentUsage.TotalTokens,
				"running_total":     recorder.stats.TotalTokens,
			})

			contentData, _ := json.Marshal(map[string]string{"body": strings.TrimSpace(contentResp)})
			if _, err := h.Queries.CreateCourseItemWithGroup(ctx, database.CreateCourseItemWithGroupParams{
				CourseID:  courseID,
				ModuleID:  pgtype.Int8{Int64: dbMod.ID, Valid: true},
				ItemType:  "content",
				SortOrder: int32(itemsCreated),
				Data:      contentData,
			}); err != nil {
				log.Printf("[mock_course] content item persist (%s): %v", secLabel, err)
			}
			itemsCreated++

			// 2b. Quiz items for the section.
			for qi := 0; qi < itemsPerSection; qi++ {
				qLabel := fmt.Sprintf("%s:question-%d", secLabel, qi+1)
				qPrompt := fmt.Sprintf(
					`Write one multiple-choice question about the following section content. Section: "%s". Module: "%s".`,
					firstSentence(strings.TrimSpace(contentResp)), mod.Title,
				)
				qResp, qUsage, err := h.LLM.ChatWithUsage(mockQuestionSystem, qPrompt)
				if err != nil {
					return recorder.stats, fmt.Errorf("question LLM call (%s): %w", qLabel, err)
				}
				recorder.record(ctx, qLabel, qUsage)
				sw.send("tokens", gin.H{
					"label":             qLabel,
					"prompt_tokens":     qUsage.PromptTokens,
					"completion_tokens": qUsage.CompletionTokens,
					"total_tokens":      qUsage.TotalTokens,
					"running_total":     recorder.stats.TotalTokens,
				})

				// Persist even if the JSON is malformed — the raw text goes
				// into `data` so the run still counts toward token totals.
				itemData := normalizeMockQuestion(qResp)
				if _, err := h.Queries.CreateCourseItemWithGroup(ctx, database.CreateCourseItemWithGroupParams{
					CourseID:  courseID,
					ModuleID:  pgtype.Int8{Int64: dbMod.ID, Valid: true},
					ItemType:  "mc",
					SortOrder: int32(itemsCreated),
					Data:      itemData,
				}); err != nil {
					log.Printf("[mock_course] question item persist (%s): %v", qLabel, err)
				}
				itemsCreated++
				recorder.stats.ItemsCreated++
			}
		}
	}

	recorder.stats.DurationSeconds = time.Since(start).Seconds()
	sw.send("step", gin.H{"detail": fmt.Sprintf("Done in %.1fs", recorder.stats.DurationSeconds)})
	return recorder.stats, nil
}

// mockOutlineModule is the parsed shape returned by the outline LLM call.
type mockOutlineModule struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// parseMockOutline tolerates partial / malformed JSON by falling back to
// deterministic module titles. We never want a flaky model response to abort
// a spend-benchmark run halfway through.
func parseMockOutline(raw string, want int, topic string) []mockOutlineModule {
	cleaned := stripMarkdownFences(strings.TrimSpace(raw))
	var parsed struct {
		Title       string              `json:"title"`
		Description string              `json:"description"`
		Modules     []mockOutlineModule `json:"modules"`
	}
	if err := json.Unmarshal([]byte(cleaned), &parsed); err == nil && len(parsed.Modules) > 0 {
		if len(parsed.Modules) > want {
			parsed.Modules = parsed.Modules[:want]
		}
		return parsed.Modules
	}
	// Fallback: deterministic shape so the rest of the pipeline has modules
	// to iterate even if the model returned prose instead of JSON.
	out := make([]mockOutlineModule, want)
	for i := 0; i < want; i++ {
		out[i] = mockOutlineModule{
			Title:       fmt.Sprintf("%s — Part %d", topic, i+1),
			Description: fmt.Sprintf("Module %d of the mock course on %s.", i+1, topic),
		}
	}
	return out
}

// normalizeMockQuestion turns an LLM response into valid JSONB for course_items.
// If the model didn't return parseable JSON, we wrap the raw text as the stem
// so the item still persists (and the tokens still count).
func normalizeMockQuestion(raw string) []byte {
	cleaned := stripMarkdownFences(strings.TrimSpace(raw))
	var q struct {
		Stem        string   `json:"stem"`
		Options     []string `json:"options"`
		AnswerIndex int      `json:"answerIndex"`
		Explanation string   `json:"explanation"`
	}
	if err := json.Unmarshal([]byte(cleaned), &q); err == nil && q.Stem != "" && len(q.Options) >= 2 {
		if q.AnswerIndex < 0 || q.AnswerIndex >= len(q.Options) {
			q.AnswerIndex = 0
		}
		b, _ := json.Marshal(q)
		return b
	}
	b, _ := json.Marshal(map[string]any{
		"stem":        firstSentence(raw),
		"options":     []string{"A", "B", "C", "D"},
		"answerIndex": 0,
		"explanation": "(raw response preserved)",
		"raw":         raw,
	})
	return b
}

func firstSentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	for i, r := range s {
		if r == '.' || r == '!' || r == '\n' {
			return strings.TrimSpace(s[:i+1])
		}
	}
	if len(s) > 240 {
		return s[:240] + "…"
	}
	return s
}
