package middlewares

import (
	database "fgb-lp/database/queries"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RequireAuth(queries *database.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("session_token")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		session, err := queries.GetSessionByToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
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

	rows, err := queries.GetDB().Query(c.Request.Context(),
		"SELECT role FROM user_roles WHERE user_id = $1", userID)
	if err != nil {
		// Fall back to single role
		return []string{user.Role}, nil
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			continue
		}
		roles = append(roles, r)
	}

	if len(roles) == 0 {
		return []string{user.Role}, nil
	}

	return roles, nil
}
