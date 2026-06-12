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
		c.Next()
	}
}
