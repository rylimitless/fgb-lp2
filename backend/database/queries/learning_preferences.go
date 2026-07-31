package database

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"
)

// LearningPreferenceTypes holds the adaptive-by-type fields added by a
// runMigrations ALTER TABLE after sqlc codegen: an explicit preferred_type
// override and learned per-type accuracy in type_stats.
//
// type_stats shape (JSONB): { "<item_type>": {"answered": int, "correct": int} }
type LearningPreferenceTypes struct {
	UserID        int64
	Theta         float64
	PreferredType pgtype.Text // nil/Valid=false → no explicit override
	TypeStats     []byte      // raw JSONB; nil → "{}"
}

// GetLearningPreferenceTypes loads theta + the type-tracking columns. Falls
// back to theta=0 / empty stats if the row is missing.
func (q *Queries) GetLearningPreferenceTypes(ctx context.Context, userID int64) (LearningPreferenceTypes, error) {
	row := q.db.QueryRow(ctx,
		`select user_id, coalesce(theta, 0.0), preferred_type, coalesce(type_stats, '{}'::jsonb)
		   from learning_preferences where user_id = $1`,
		userID,
	)
	var p LearningPreferenceTypes
	err := row.Scan(&p.UserID, &p.Theta, &p.PreferredType, &p.TypeStats)
	return p, err
}

// RecordTypeOutcome bumps the answered/correct counters for one item_type
// inside the learner's type_stats jsonb and returns the updated document.
// Uses COALESCE so a missing sub-object starts from 0. Read-modify-write is
// acceptable here because a learner has at most one active adaptive session at
// a time, so writes for a given user are not truly concurrent.
func (q *Queries) RecordTypeOutcome(ctx context.Context, userID int64, itemType string, correct bool) ([]byte, error) {
	// Pull current counters for this type (defaulting to 0/0), then write back.
	var stats []byte
	if err := q.db.QueryRow(ctx,
		`select coalesce(type_stats, '{}'::jsonb) from learning_preferences where user_id = $1`,
		userID,
	).Scan(&stats); err != nil {
		return nil, err
	}

	var doc map[string]map[string]int
	if err := json.Unmarshal(stats, &doc); err != nil || doc == nil {
		doc = map[string]map[string]int{}
	}
	entry := doc[itemType]
	if entry == nil {
		entry = map[string]int{"answered": 0, "correct": 0}
	}
	entry["answered"]++
	if correct {
		entry["correct"]++
	}
	doc[itemType] = entry

	updated, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	if _, err := q.db.Exec(ctx,
		`update learning_preferences set type_stats = $2, updated_at = now() where user_id = $1`,
		userID, updated,
	); err != nil {
		return nil, err
	}
	return updated, nil
}

// SetPreferredType sets the explicit preferred question-type override for a
// learner. Pass empty string to clear it (engine then falls back to learned
// best-performing type).
func (q *Queries) SetPreferredType(ctx context.Context, userID int64, itemType string) error {
	var v pgtype.Text
	if itemType != "" {
		v = pgtype.Text{String: itemType, Valid: true}
	}
	// Upsert the row first (in case it doesn't exist) then set the column.
	_, err := q.db.Exec(ctx,
		`insert into learning_preferences (user_id) values ($1)
		 on conflict (user_id) do nothing`,
		userID,
	)
	if err != nil {
		return err
	}
	_, err = q.db.Exec(ctx,
		`update learning_preferences set preferred_type = $2, updated_at = now() where user_id = $1`,
		userID, v,
	)
	return err
}
