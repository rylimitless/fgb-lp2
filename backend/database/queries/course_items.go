package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// CourseItemWithGroup mirrors CourseItem but adds question_group_id, which was
// added by a runMigrations ALTER TABLE after sqlc codegen. Items that test the
// SAME concept in different formats share a group id; the adaptive engine uses
// this to serve each learner their preferred/best question type per concept.
type CourseItemWithGroup struct {
	ID              int64
	CourseID        int64
	ModuleID        pgtype.Int8
	ItemType        string
	SortOrder       int32
	Data            []byte
	QuestionGroupID pgtype.Text // nil/Invalid = not part of a variant group
	CreatedAt       string
}

// CreateCourseItemWithGroupParams is CreateCourseItemParams plus the optional
// question_group_id used to link question-type variants of one concept.
type CreateCourseItemWithGroupParams struct {
	CourseID        int64
	ModuleID        pgtype.Int8
	ItemType        string
	SortOrder       int32
	Data            []byte
	QuestionGroupID string // "" → store NULL (no group, e.g. content items)
}

// CreateCourseItemWithGroup inserts a course item, optionally tagging it with a
// question_group_id so its type-variants can be linked for adaptive selection.
// Use this for generated assessment items; plain content items can pass "".
func (q *Queries) CreateCourseItemWithGroup(ctx context.Context, arg CreateCourseItemWithGroupParams) (CourseItemWithGroup, error) {
	var groupID pgtype.Text
	if arg.QuestionGroupID != "" {
		groupID = pgtype.Text{String: arg.QuestionGroupID, Valid: true}
	}
	row := q.db.QueryRow(ctx,
		`insert into course_items (course_id, module_id, item_type, sort_order, data, question_group_id)
		 values ($1, $2, $3, $4, $5, $6)
		 returning id, course_id, module_id, item_type, sort_order, data,
		           question_group_id, created_at::text`,
		arg.CourseID, arg.ModuleID, arg.ItemType, arg.SortOrder, arg.Data, groupID,
	)
	var i CourseItemWithGroup
	err := row.Scan(
		&i.ID, &i.CourseID, &i.ModuleID, &i.ItemType, &i.SortOrder,
		&i.Data, &i.QuestionGroupID, &i.CreatedAt,
	)
	return i, err
}

// GetCourseItemsByCourseWithGroup lists a course's items (sorted) including the
// question_group_id column. Used by the adaptive engine so it can group
// concept variants together.
func (q *Queries) GetCourseItemsByCourseWithGroup(ctx context.Context, courseID int64) ([]CourseItemWithGroup, error) {
	rows, err := q.db.Query(ctx,
		`select id, course_id, module_id, item_type, sort_order, data,
		        question_group_id, created_at::text
		   from course_items where course_id = $1 order by sort_order asc`,
		courseID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CourseItemWithGroup
	for rows.Next() {
		var i CourseItemWithGroup
		if err := rows.Scan(
			&i.ID, &i.CourseID, &i.ModuleID, &i.ItemType, &i.SortOrder,
			&i.Data, &i.QuestionGroupID, &i.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}
