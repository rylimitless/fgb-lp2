package ai

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	database "fgb-lp/database/queries"

	"github.com/gin-gonic/gin"
)

// ---- Developer options: mock course generator + token spend dashboard ----
//
// These routes are admin-only (registered against adminGroup in main.go via
// RegisterDevRoutes) because they incur real LLM cost and create real course
// rows. The mock generator is intentionally simple — see mock.go for why.

// ListMockPresets returns the canonical preset menu the UI renders.
func (h *Handler) ListMockPresets(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"presets": MockPresets})
}

// GenerateMockCourse streams a mock-course generation run over SSE. Unlike the
// real course pipeline (which enqueues and streams from a worker), this runs
// inline: each LLM call is awaited so its token usage can be reported the
// instant the provider returns it. Runs are short (typically <2 min) and
// single-user, so the simpler model fits.
//
// Query params:
//
//	preset = tiny | small | medium | large (default: small)
//
// The connection stays open for the duration of the run; the final `done`
// event carries the MockRunStats aggregate.
func (h *Handler) GenerateMockCourse(c *gin.Context) {
	userIDVal, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID, ok := userIDVal.(int64)
	if !ok || userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user session"})
		return
	}

	presetID := strings.TrimSpace(c.DefaultQuery("preset", "small"))
	preset := LookupPreset(presetID)
	if preset == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown preset; valid: tiny, small, medium, large"})
		return
	}

	w, ok := newSSEWriter(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	jobRef := fmt.Sprintf("mock-%s-%d", preset.ID, time.Now().UnixNano())

	// Cap the whole run at 8 minutes so a stuck model can't hold an admin's
	// SSE connection open forever.
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Minute)
	defer cancel()

	w.send("step", gin.H{"detail": fmt.Sprintf("Starting mock course (preset=%s)", preset.Label)})

	stats, err := h.runMockCourse(ctx, *preset, jobRef, userID, w)
	if err != nil {
		// ctx may be cancelled by the client closing the tab; log + send error.
		w.sendError(fmt.Sprintf("mock course failed: %v", err))
		return
	}

	w.send("done", gin.H{
		"job_id":               stats.JobID,
		"preset_id":            stats.PresetID,
		"course_id":            stats.CourseID,
		"model":                stats.Model,
		"llm_calls":            stats.LLMCalls,
		"prompt_tokens":        stats.PromptTokens,
		"completion_tokens":    stats.CompletionTokens,
		"total_tokens":         stats.TotalTokens,
		"cached_prompt_tokens": stats.CachedPrompt,
		"reasoning_tokens":     stats.ReasoningTokens,
		"duration_seconds":     stats.DurationSeconds,
		"modules_created":      stats.ModulesCreated,
		"items_created":        stats.ItemsCreated,
	})
}

// LLMTokenUsageResponse is the payload returned by GetLLMTokenUsage. Costs are
// NOT computed server-side — pricing changes often and varies by deployment,
// so the UI computes spend from the token counts using configurable rates.
type LLMTokenUsageResponse struct {
	BySource []database.LLMTokenUsageAggregate `json:"by_source"`
	ByJob    []database.LLMTokenUsageAggregate `json:"by_job"`
	Recent   []database.LLMTokenUsageRow       `json:"recent"`
	Totals   database.LLMTokenUsageAggregate   `json:"totals"`
	MockJobs int64                             `json:"mock_jobs"`
}

// GetLLMTokenUsage returns aggregate + recent token usage for the spend
// dashboard. `mock_jobs` is the count of distinct mock-course runs, used by
// the UI to compute "average spend per mock course".
func (h *Handler) GetLLMTokenUsage(c *gin.Context) {
	limit := 100

	ctx := c.Request.Context()

	bySource, err := h.Queries.AggregateLLMTokenUsage(ctx, int32(limit))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	byJob, err := h.Queries.AggregateLLMTokenUsageByJob(ctx, "mock_course", 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	recent, err := h.Queries.RecentLLMTokenUsage(ctx, "", 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Headline totals across everything tracked.
	var totals database.LLMTokenUsageAggregate
	totals.Source = "all"
	for _, s := range bySource {
		totals.Calls += s.Calls
		totals.PromptTokens += s.PromptTokens
		totals.CompletionTokens += s.CompletionTokens
		totals.TotalTokens += s.TotalTokens
		totals.CachedPromptTokens += s.CachedPromptTokens
		totals.ReasoningTokens += s.ReasoningTokens
	}

	// Count distinct mock-course runs for the "average per course" stat.
	_, _, _, _, mockJobs, err := h.Queries.SumLLMTokenUsageByJob(ctx, "mock_course")
	if err != nil {
		mockJobs = int64(len(byJob))
	}

	c.JSON(http.StatusOK, LLMTokenUsageResponse{
		BySource: bySource,
		ByJob:    byJob,
		Recent:   recent,
		Totals:   totals,
		MockJobs: mockJobs,
	})
}

// RegisterDevRoutes wires up the admin-only developer-tools routes onto the
// given (admin-gated) router group.
func (h *Handler) RegisterDevRoutes(r *gin.RouterGroup) {
	r.GET("/dev/token-usage", h.GetLLMTokenUsage)
	r.GET("/dev/mock-presets", h.ListMockPresets)
	r.GET("/dev/mock-course/generate", h.GenerateMockCourse)
}
