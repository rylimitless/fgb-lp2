package middlewares

import (
	"fgb-lp/authtrace"
	database "fgb-lp/database/queries"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func RequireAuth(queries *database.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("session_token")
		if err != nil {
			reason := authtrace.Classify(c.Request.Context(), queries, "")
			authtrace.Log(queries, c, "backend", "auth_401", authtrace.Prefix(""), map[string]any{
				"reason": string(reason),
			})
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		session, err := queries.GetSessionByToken(c.Request.Context(), token)
		if err != nil {
			reason := authtrace.Classify(c.Request.Context(), queries, token)
			authtrace.Log(queries, c, "backend", "auth_401", authtrace.Prefix(token), map[string]any{
				"reason":     string(reason),
				"session_id": session.ID, // zero on lookup error
			})
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Sliding renewal: when the session is within 6 hours of expiring,
		// extend it (TouchSession's WHERE clause makes this a no-op otherwise,
		// so normal requests never pay a DB write). Active users therefore
		// can't be cut off mid-task by the session lifetime.
		if time.Until(session.ExpiresAt.Time) < 6*time.Hour {
			if qerr := queries.TouchSession(c.Request.Context(), token); qerr == nil {
				authtrace.Log(queries, c, "backend", "session_touched", authtrace.Prefix(token), map[string]any{
					"user_id":    session.UserID,
					"session_id": session.ID,
				})
			}
		}

		c.Set("user_id", session.UserID)

		// Load user roles from user_roles table into context
		roles, err := GetUserRoles(c, queries, session.UserID)
		if err == nil {
			c.Set("user_roles", roles)
		}

		c.Next()
	}
}

// GetUserRoles fetches roles for a user from the user_roles junction table.
// Falls back to the single role column if user_roles is empty.
func GetUserRoles(c *gin.Context, queries *database.Queries, userID int64) ([]string, error) {
	user, err := queries.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		return nil, err
	}

	roles, err := queries.GetUserRoles(c.Request.Context(), userID)
	if err != nil || len(roles) == 0 {
		// Fall back to single role
		return []string{user.Role}, nil
	}
	return roles, nil
}
