package database

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pgvector/pgvector-go"
)

// ModuleWithStatus mirrors Module but is updated by hand to include the
// `status`, `question_types`, and `limitations` columns added by the staged
// course builder migrations. sqlc codegen doesn't know about these columns
// because they were added via ALTER TABLE in runMigrations rather than in
// the parsed schema at codegen time.
type ModuleWithStatus struct {
	ID                  int64
	CourseID            int64
	Title               string
	Description         string
	SortOrder           int32
	Status              string
	QuestionTypes       []string // nil/empty = allow all types
	QuestionTypeTargets map[string]int
	Limitations         []map[string]any // AI-reported skips: {type, reason, section}
	CreatedAt           string           // timestamptz as text
}

// scanModule scans a module row whose SELECT projects the staged-builder
// columns in the canonical order used by GetModule / GetModulesByCourseWithStatus.
func scanModule(scanner interface {
	Scan(dest ...any) error
}) (ModuleWithStatus, error) {
	var (
		m                   ModuleWithStatus
		questionTypes       []byte
		questionTypeTargets []byte
		limitations         []byte
	)
	if err := scanner.Scan(
		&m.ID, &m.CourseID, &m.Title, &m.Description, &m.SortOrder,
		&m.Status, &questionTypes, &questionTypeTargets, &limitations, &m.CreatedAt,
	); err != nil {
		return ModuleWithStatus{}, err
	}
	if len(questionTypes) > 0 {
		_ = json.Unmarshal(questionTypes, &m.QuestionTypes)
	}
	if len(questionTypeTargets) > 0 {
		_ = json.Unmarshal(questionTypeTargets, &m.QuestionTypeTargets)
	}
	if m.QuestionTypeTargets == nil {
		m.QuestionTypeTargets = map[string]int{}
	}
	if len(limitations) > 0 {
		_ = json.Unmarshal(limitations, &m.Limitations)
	}
	if m.Limitations == nil {
		m.Limitations = []map[string]any{}
	}
	return m, nil
}

// moduleSelectCols is the canonical column projection for staged-builder
// module queries. Keep in sync with scanModule.
const moduleSelectCols = `id, course_id, title, description, sort_order,
       COALESCE(status, 'pending'),
       question_types::text,
	       COALESCE(question_type_targets::text, '{}'),
       COALESCE(limitations::text, 'null'),
       created_at::text`

// GetModule fetches a single module row by ID, including its generation status
// and staged-builder metadata (question types, limitations).
func (q *Queries) GetModule(ctx context.Context, id int64) (*ModuleWithStatus, error) {
	row := q.db.QueryRow(ctx,
		`SELECT `+moduleSelectCols+` FROM modules WHERE id = $1`,
		id,
	)
	m, err := scanModule(row)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetModulesByCourseWithStatus lists all modules for a course, including
// status, question types, and limitations.
func (q *Queries) GetModulesByCourseWithStatus(ctx context.Context, courseID int64) ([]ModuleWithStatus, error) {
	rows, err := q.db.Query(ctx,
		`SELECT `+moduleSelectCols+`
		 FROM modules WHERE course_id = $1 ORDER BY sort_order ASC, id ASC`,
		courseID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ModuleWithStatus
	for rows.Next() {
		m, err := scanModule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (q *Queries) SearchDocumentChunksByIDs(ctx context.Context, embedding pgvector.Vector, documentIDs []int64, limit int32) ([]SearchDocumentChunksRow, error) {
	rows, err := q.db.Query(ctx,
		`SELECT dc.id, dc.document_id, dc.chunk_index, dc.content, dc.page_number,
		        dc.source_label, dc.embedding, dc.created_at, d.title AS document_title
		   FROM document_chunks dc
		   JOIN documents d ON d.id = dc.document_id
		  WHERE d.approved = true
		    AND dc.embedding IS NOT NULL
		    AND dc.document_id = ANY($2::bigint[])
		  ORDER BY dc.embedding <=> $1
		  LIMIT $3`,
		embedding, documentIDs, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []SearchDocumentChunksRow
	for rows.Next() {
		var chunk SearchDocumentChunksRow
		if err := rows.Scan(
			&chunk.ID, &chunk.DocumentID, &chunk.ChunkIndex, &chunk.Content,
			&chunk.PageNumber, &chunk.SourceLabel, &chunk.Embedding, &chunk.CreatedAt,
			&chunk.DocumentTitle,
		); err != nil {
			return nil, err
		}
		chunks = append(chunks, chunk)
	}
	return chunks, rows.Err()
}

func (q *Queries) UpdateModuleQuestionTypeTargets(ctx context.Context, id int64, targets []byte) error {
	_, err := q.db.Exec(ctx,
		`UPDATE modules SET question_type_targets = $2::jsonb WHERE id = $1`,
		id, targets,
	)
	return err
}

// UpdateModuleStatus sets the generation status for a single module row.
func (q *Queries) UpdateModuleStatus(ctx context.Context, id int64, status string) error {
	_, err := q.db.Exec(ctx,
		`UPDATE modules SET status = $2 WHERE id = $1`,
		id, status,
	)
	return err
}

// UpdateModuleMeta updates a module's editable outline fields (title,
// description, sort_order, question_types). Used by the staged builder when
// the user edits the outline before any content is generated.
// Pass questionTypes=nil to leave the existing allowlist unchanged.
func (q *Queries) UpdateModuleMeta(ctx context.Context, id int64, title, description string, sortOrder int32, questionTypes []byte) error {
	// When questionTypes is nil we use COALESCE to preserve the existing value.
	if questionTypes != nil {
		_, err := q.db.Exec(ctx,
			`UPDATE modules
			   SET title = $2, description = $3, sort_order = $4, question_types = $5::jsonb
			 WHERE id = $1`,
			id, title, description, sortOrder, questionTypes,
		)
		return err
	}
	_, err := q.db.Exec(ctx,
		`UPDATE modules
		   SET title = $2, description = $3, sort_order = $4
		 WHERE id = $1`,
		id, title, description, sortOrder,
	)
	return err
}

// UpdateModuleQuestionTypes sets only the question-type allowlist for a module.
// Useful when the user changes types without touching other outline fields.
func (q *Queries) UpdateModuleQuestionTypes(ctx context.Context, id int64, questionTypes []byte) error {
	_, err := q.db.Exec(ctx,
		`UPDATE modules SET question_types = $2::jsonb WHERE id = $1`,
		questionTypes, id,
	)
	return err
}

// UpdateModuleLimitations stores AI-reported question-type skips for a module.
// Called by the worker after generation completes.
func (q *Queries) UpdateModuleLimitations(ctx context.Context, id int64, limitations []byte) error {
	_, err := q.db.Exec(ctx,
		`UPDATE modules SET limitations = $2::jsonb WHERE id = $1`,
		limitations, id,
	)
	return err
}

// DeleteModule removes a single module and (via ON DELETE SET NULL on
// course_items.module_id) detaches its items. Items are kept on the course so
// a fat-finger outline edit doesn't silently destroy generated content; the
// UI is responsible for also calling DeleteCourseItemsByModule when a true
// purge is wanted.
func (q *Queries) DeleteModule(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, `DELETE FROM modules WHERE id = $1`, id)
	return err
}

// DeleteModules removes multiple modules (scoped to a course) in a single
// statement. Like DeleteModule it only detaches items via ON DELETE SET NULL
// — call DeleteCourseItemsByModules first if a true purge is wanted.
func (q *Queries) DeleteModules(ctx context.Context, courseID int64, moduleIDs []int64) error {
	if len(moduleIDs) == 0 {
		return nil
	}
	_, err := q.db.Exec(ctx,
		`DELETE FROM modules WHERE course_id = $1 AND id = ANY($2::bigint[])`,
		courseID, moduleIDs,
	)
	return err
}

// DeleteCourseItemsByModules hard-deletes every item attached to any of the
// given modules. Batch counterpart to DeleteCourseItemsByModule.
func (q *Queries) DeleteCourseItemsByModules(ctx context.Context, moduleIDs []int64) error {
	if len(moduleIDs) == 0 {
		return nil
	}
	_, err := q.db.Exec(ctx,
		`DELETE FROM course_items WHERE module_id = ANY($1::bigint[])`,
		moduleIDs,
	)
	return err
}

// DeleteCourseItemsByModule hard-deletes every item attached to a module.
// Used by module regeneration so we don't accumulate stale items.
func (q *Queries) DeleteCourseItemsByModule(ctx context.Context, moduleID pgtype.Int8) error {
	_, err := q.db.Exec(ctx, `DELETE FROM course_items WHERE module_id = $1`, moduleID)
	return err
}

// ReorderModules reassigns sort_order for every module in `moduleIDs` based
// on the array position. Modules not in the list keep their existing order.
// Uses unnest WITH ORDINALITY so it's a single statement — atomic and fast
// even for large module lists.
func (q *Queries) ReorderModules(ctx context.Context, courseID int64, moduleIDs []int64) error {
	if len(moduleIDs) == 0 {
		return nil
	}
	// unnest WITH ORDINALITY returns (value, ordinal) columns. We alias them
	// explicitly (t.elem, t.ord) because the default column names from
	// unnest are not stable across PostgreSQL versions — referencing
	// "id" or "ordinality" directly fails.
	_, err := q.db.Exec(ctx,
		`UPDATE modules SET sort_order = sub.new_order
		   FROM (SELECT t.elem AS id, t.ord::int - 1 AS new_order
		           FROM unnest($2::bigint[]) WITH ORDINALITY AS t(elem, ord)) AS sub
		  WHERE modules.id = sub.id AND modules.course_id = $1`,
		courseID, moduleIDs,
	)
	return err
}

// MarkCourseModulesPending resets every module of a course back to 'pending'.
// Used if the user wants to regenerate the whole course module-by-module after
// editing the outline.
func (q *Queries) MarkCourseModulesPending(ctx context.Context, courseID int64) error {
	_, err := q.db.Exec(ctx,
		`UPDATE modules SET status = 'pending' WHERE course_id = $1`,
		courseID,
	)
	return err
}
