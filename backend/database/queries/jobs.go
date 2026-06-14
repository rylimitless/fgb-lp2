package database

import (
	"context"
	"encoding/json"
)

// GenerationJobRow mirrors ai.GenerationJob for DB storage.
type GenerationJobRow struct {
	ID        string
	Status    string
	Request   json.RawMessage
	Steps     json.RawMessage
	Modules   json.RawMessage
	Result    json.RawMessage
	Error     string
	CourseID  *int64
	CreatedAt string // timestamptz returned as string by pgx
}

// CreateGenerationJob inserts a new pending job.
func (q *Queries) CreateGenerationJob(ctx context.Context, id string, request []byte) error {
	_, err := q.db.Exec(ctx,
		`INSERT INTO course_generation_jobs (id, status, request) VALUES ($1, 'pending', $2)`,
		id, request,
	)
	return err
}

// StartGenerationJob marks a job as running.
func (q *Queries) StartGenerationJob(ctx context.Context, id string) error {
	_, err := q.db.Exec(ctx,
		`UPDATE course_generation_jobs SET status = 'running', updated_at = now() WHERE id = $1`,
		id,
	)
	return err
}

// UpdateGenerationJobProgress updates steps and modules arrays.
func (q *Queries) UpdateGenerationJobProgress(ctx context.Context, id string, steps, modules []byte) error {
	_, err := q.db.Exec(ctx,
		`UPDATE course_generation_jobs SET steps = $2, modules = $3, updated_at = now() WHERE id = $1`,
		id, steps, modules,
	)
	return err
}

// CompleteGenerationJob marks a job as completed with the result and course ID.
func (q *Queries) CompleteGenerationJob(ctx context.Context, id string, result []byte, courseID int64) error {
	_, err := q.db.Exec(ctx,
		`UPDATE course_generation_jobs SET status = 'completed', result = $3, course_id = $4, updated_at = now() WHERE id = $1 AND status = 'running'`,
		id, result, courseID,
	)
	return err
}

// FailGenerationJob marks a job as failed.
func (q *Queries) FailGenerationJob(ctx context.Context, id string, errMsg string) error {
	_, err := q.db.Exec(ctx,
		`UPDATE course_generation_jobs SET status = 'failed', error = $2, updated_at = now() WHERE id = $1`,
		id, errMsg,
	)
	return err
}

// GetGenerationJob fetches a single job by ID.
func (q *Queries) GetGenerationJob(ctx context.Context, id string) (*GenerationJobRow, error) {
	row := q.db.QueryRow(ctx,
		`SELECT id, status, request, steps, modules, result, COALESCE(error, ''), course_id, created_at::text FROM course_generation_jobs WHERE id = $1`,
		id,
	)
	var j GenerationJobRow
	err := row.Scan(&j.ID, &j.Status, &j.Request, &j.Steps, &j.Modules, &j.Result, &j.Error, &j.CourseID, &j.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &j, nil
}

// ListActiveGenerationJobs returns jobs that are pending or running.
func (q *Queries) ListActiveGenerationJobs(ctx context.Context) ([]*GenerationJobRow, error) {
	rows, err := q.db.Query(ctx,
		`SELECT id, status, request, steps, modules, COALESCE(result::text, 'null')::jsonb, COALESCE(error, ''), course_id, created_at::text FROM course_generation_jobs WHERE status IN ('pending', 'running') ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*GenerationJobRow
	for rows.Next() {
		var j GenerationJobRow
		if err := rows.Scan(&j.ID, &j.Status, &j.Request, &j.Steps, &j.Modules, &j.Result, &j.Error, &j.CourseID, &j.CreatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, &j)
	}
	return jobs, rows.Err()
}

// ResetStaleJobs marks any "running" jobs as "pending" so they get re-enqueued on restart.
func (q *Queries) ResetStaleJobs(ctx context.Context) error {
	_, err := q.db.Exec(ctx,
		`UPDATE course_generation_jobs SET status = 'pending', updated_at = now() WHERE status = 'running'`,
	)
	return err
}
