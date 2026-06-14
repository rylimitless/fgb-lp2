package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireContentCreatorOrAdmin wraps a handler to require content creator or admin role.
// Use this for individual route handlers that need content creation privileges.
func WrapRequireRole(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc {
	checker := RequireRole(roles...)
	return func(c *gin.Context) {
		// Store current handlers
		checker(c)
		if c.IsAborted() {
			return
		}
		handler(c)
	}
}

// RequireAnyRole is a middleware that checks if the user has any of the specified roles.
// This is an alias kept for backward compatibility.
func RequireAnyRole(roles ...string) gin.HandlerFunc {
	return RequireRole(roles...)
}

// GetUserRolesFromContext extracts user roles from the gin context.
func GetUserRolesFromContext(c *gin.Context) []string {
	roles, exists := c.Get("user_roles")
	if !exists {
		return nil
	}
	rolesList, ok := roles.([]string)
	if !ok {
		return nil
	}
	return rolesList
}

// UserHasRole checks if the current user has a specific role.
func UserHasRole(c *gin.Context, role string) bool {
	roles := GetUserRolesFromContext(c)
	for _, r := range roles {
		if r == role || r == "admin" {
			return true
		}
	}
	return false
}

// UserCanManageContent checks if user is a content creator or admin.
func UserCanManageContent(c *gin.Context) bool {
	return UserHasRole(c, "content creator")
}

// RequireContentManagement returns 403 if user is not a content creator or admin.
func RequireContentManagement(c *gin.Context) {
	if !UserCanManageContent(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Content creator access required"})
		c.Abort()
	}
}
