package middlewares

import (
	"github.com/gin-gonic/gin"
)

func RequireAdmin() gin.HandlerFunc {
	return RequireRole("admin")
}

// RequireContentCreator checks for content creator (or admin) role
func RequireContentCreator() gin.HandlerFunc {
	return RequireRole("content creator")
}

// RequireApprover checks for approver (or admin) role
func RequireApprover() gin.HandlerFunc {
	return RequireRole("approver")
}
