package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireRole checks that the authenticated user has at least one of the
// specified roles. Admin users always pass through.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles, exists := c.Get("user_roles")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			c.Abort()
			return
		}

		rolesList, ok := userRoles.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			c.Abort()
			return
		}

		// Admin always has access
		for _, r := range rolesList {
			if r == "admin" {
				c.Next()
				return
			}
		}

		// Check if user has any of the required roles
		roleSet := make(map[string]bool, len(rolesList))
		for _, r := range rolesList {
			roleSet[r] = true
		}
		for _, required := range roles {
			if roleSet[required] {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		c.Abort()
	}
}
