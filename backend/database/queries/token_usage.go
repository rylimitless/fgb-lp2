package database

import (
	"context"
	"time"
)

// LLMTokenUsageRow is one persisted chat-completion record. Every call to the
// model (outline, module, coach, discovery, mock_course, ...) writes one of
// these so the developer-tools spend dashboard can aggregate over them. The
// provider's own accounting is stored verbatim — no estimation.
type LLMTokenUsageRow struct {
	ID                 int64
	Model              string
	Source             string // mock_course | course_builder | coach | discovery | ...
	Label              string // free-form, e.g. "outline" or "module-3:section-1:questions"
	PromptTokens       int64
	CompletionTokens   int64
	TotalTokens        int64
	CachedPromptTokens int64
	ReasoningTokens    int64
	CourseID           *int64
	UserID             *int64
	JobRef             *string
	CreatedAt          time.Time
}

// LogLLMTokenUsageParams is the payload for LogLLMTokenUsage.
type LogLLMTokenUsageParams struct {
	Model              string
	Source             string
	Label              string
	PromptTokens       int64
	CompletionTokens   int64
	TotalTokens        int64
	CachedPromptTokens int64
	ReasoningTokens    int64
	CourseID           *int64 // nil = not associated with a course
	UserID             *int64 // nil = system / unattributed
	JobRef             *string
}

// LogLLMTokenUsage inserts one token-usage row. Called after every LLM call
// that wants its spend tracked.
func (q *Queries) LogLLMTokenUsage(ctx context.Context, arg LogLLMTokenUsageParams) (LLMTokenUsageRow, error) {
	row := q.db.QueryRow(ctx,
		`insert into llm_token_usage
		   (model, source, label, prompt_tokens, completion_tokens, total_tokens,
		    cached_prompt_tokens, reasoning_tokens, course_id, user_id, job_ref)
		 values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 returning id, model, source, label, prompt_tokens, completion_tokens,
		           total_tokens, cached_prompt_tokens, reasoning_tokens, course_id,
		           user_id, job_ref, created_at`,
		arg.Model, arg.Source, arg.Label, arg.PromptTokens, arg.CompletionTokens,
		arg.TotalTokens, arg.CachedPromptTokens, arg.ReasoningTokens, arg.CourseID,
		arg.UserID, arg.JobRef,
	)
	var u LLMTokenUsageRow
	err := row.Scan(
		&u.ID, &u.Model, &u.Source, &u.Label, &u.PromptTokens, &u.CompletionTokens,
		&u.TotalTokens, &u.CachedPromptTokens, &u.ReasoningTokens, &u.CourseID,
		&u.UserID, &u.JobRef, &u.CreatedAt,
	)
	return u, err
}

// LLMTokenUsageAggregate is the rolled-up view the spend dashboard renders.
// We compute this server-side so the UI gets one row per source/label bucket
// instead of pulling every raw call.
type LLMTokenUsageAggregate struct {
	Source             string `json:"source"`
	Calls              int64  `json:"calls"`
	PromptTokens       int64  `json:"prompt_tokens"`
	CompletionTokens   int64  `json:"completion_tokens"`
	TotalTokens        int64  `json:"total_tokens"`
	CachedPromptTokens int64  `json:"cached_prompt_tokens"`
	ReasoningTokens    int64  `json:"reasoning_tokens"`
}

// AggregateLLMTokenUsage groups all token-usage rows by source. Pass 0 for
// limit to get all sources.
func (q *Queries) AggregateLLMTokenUsage(ctx context.Context, limit int32) ([]LLMTokenUsageAggregate, error) {
	rows, err := q.db.Query(ctx,
		`select source,
		        count(*)            as calls,
		        coalesce(sum(prompt_tokens), 0),
		        coalesce(sum(completion_tokens), 0),
		        coalesce(sum(total_tokens), 0),
		        coalesce(sum(cached_prompt_tokens), 0),
		        coalesce(sum(reasoning_tokens), 0)
		   from llm_token_usage
		  group by source
		  order by coalesce(sum(total_tokens), 0) desc
		  limit $1`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LLMTokenUsageAggregate
	for rows.Next() {
		var a LLMTokenUsageAggregate
		if err := rows.Scan(&a.Source, &a.Calls, &a.PromptTokens, &a.CompletionTokens,
			&a.TotalTokens, &a.CachedPromptTokens, &a.ReasoningTokens); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// AggregateLLMTokenUsageByJob groups by job_ref — used by the mock-course
// dashboard to compare one run against another.
func (q *Queries) AggregateLLMTokenUsageByJob(ctx context.Context, source string, limit int32) ([]LLMTokenUsageAggregate, error) {
	rows, err := q.db.Query(ctx,
		`select coalesce(job_ref, ''),
		        count(*)            as calls,
		        coalesce(sum(prompt_tokens), 0),
		        coalesce(sum(completion_tokens), 0),
		        coalesce(sum(total_tokens), 0),
		        coalesce(sum(cached_prompt_tokens), 0),
		        coalesce(sum(reasoning_tokens), 0)
		   from llm_token_usage
		  where source = $1
		  group by job_ref
		  order by max(created_at) desc
		  limit $2`,
		source, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LLMTokenUsageAggregate
	for rows.Next() {
		var a LLMTokenUsageAggregate
		if err := rows.Scan(&a.Source, &a.Calls, &a.PromptTokens, &a.CompletionTokens,
			&a.TotalTokens, &a.CachedPromptTokens, &a.ReasoningTokens); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// RecentLLMTokenUsage returns the most recent individual calls — used by the
// developer UI to show a live log as a mock course is generated.
func (q *Queries) RecentLLMTokenUsage(ctx context.Context, source string, limit int32) ([]LLMTokenUsageRow, error) {
	rows, err := q.db.Query(ctx,
		`select id, model, source, label, prompt_tokens, completion_tokens,
		        total_tokens, cached_prompt_tokens, reasoning_tokens, course_id,
		        user_id, job_ref, created_at
		   from llm_token_usage
		  where source = $1 or $1 = ''
		  order by created_at desc
		  limit $2`,
		source, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LLMTokenUsageRow
	for rows.Next() {
		var u LLMTokenUsageRow
		if err := rows.Scan(&u.ID, &u.Model, &u.Source, &u.Label, &u.PromptTokens,
			&u.CompletionTokens, &u.TotalTokens, &u.CachedPromptTokens,
			&u.ReasoningTokens, &u.CourseID, &u.UserID, &u.JobRef, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// SumLLMTokenUsageByJob returns the per-job totals (one row per job_ref) for a
// given source. Used to compute "average spend per mock course" by averaging
// across job_refs.
func (q *Queries) SumLLMTokenUsageByJob(ctx context.Context, source string) (calls int64, prompt int64, completion int64, total int64, jobs int64, err error) {
	row := q.db.QueryRow(ctx,
		`select coalesce(count(*), 0),
		        coalesce(sum(prompt_tokens), 0),
		        coalesce(sum(completion_tokens), 0),
		        coalesce(sum(total_tokens), 0),
		        coalesce(count(distinct job_ref), 0)
		   from llm_token_usage
		  where source = $1`,
		source,
	)
	err = row.Scan(&calls, &prompt, &completion, &total, &jobs)
	return
}
