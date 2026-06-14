package audit

import (
	"context"
	"encoding/json"
	database "fgb-lp/database/queries"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

// Log writes an audit entry. It's fire-and-forget: errors are logged but not propagated.
// Pass nil for c to log without a HTTP context (e.g. background jobs).
func Log(queries *database.Queries, c *gin.Context, action string, details map[string]any) {
	uid := userID(c)
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		log.Printf("[audit] failed to marshal details for action %q: %v", action, err)
		detailsJSON = []byte("{}")
	}

	_, err = queries.InsertAuditLog(context.Background(), database.InsertAuditLogParams{
		UserID:  uid,
		Action:  action,
		Details: detailsJSON,
	})
	if err != nil {
		log.Printf("[audit] failed to insert audit log for action %q: %v", action, err)
	}
}

// Logf is a convenience wrapper that formats a detail message into {"msg": "..."}.
func Logf(queries *database.Queries, c *gin.Context, action string, msg string, args ...any) {
	Log(queries, c, action, map[string]any{
		"msg": fmt.Sprintf(msg, args...),
	})
}

func userID(c *gin.Context) pgtype.Int8 {
	if c == nil {
		return pgtype.Int8{Valid: false}
	}
	uid, exists := c.Get("user_id")
	if !exists {
		return pgtype.Int8{Valid: false}
	}
	id, ok := uid.(int64)
	if !ok {
		return pgtype.Int8{Valid: false}
	}
	return pgtype.Int8{Int64: id, Valid: true}
}
