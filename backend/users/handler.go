package users

import (
	"fgb-lp/audit"
	database "fgb-lp/database/queries"
	"fgb-lp/functions"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Queries *database.Queries
}

func NewHandler(queries *database.Queries) *Handler {
	return &Handler{
		Queries: queries,
	}
}

func (h *Handler) ListUsers(c *gin.Context) {
	users, err := h.Queries.GetAllUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	if users == nil {
		users = []database.User{}
	}

	// Strip password hashes and include multiple roles
	type SafeUser struct {
		ID        int64    `json:"id"`
		Email     string   `json:"email"`
		Name      string   `json:"name"`
		Role      string   `json:"role"`
		Roles     []string `json:"roles"`
		CreatedAt string   `json:"created_at"`
	}
	out := make([]SafeUser, len(users))
	for i, u := range users {
		// Fetch all roles for this user
		roles := h.getUserRoles(c, u.ID)

		out[i] = SafeUser{
			ID:        u.ID,
			Email:     u.Email,
			Name:      u.Name,
			Role:      u.Role,
			Roles:     roles,
			CreatedAt: u.CreatedAt.Time.Format("2006-01-02 15:04"),
		}
	}

	c.JSON(http.StatusOK, out)
}

func (h *Handler) CreateUser(c *gin.Context) {
	var body struct {
		Email    string   `json:"email" binding:"required,email"`
		Password string   `json:"password" binding:"required,min=8"`
		Name     string   `json:"name" binding:"required"`
		Role     string   `json:"role"`
		Roles    []string `json:"roles"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Determine primary role
	primaryRole := body.Role
	if primaryRole == "" {
		if len(body.Roles) > 0 {
			primaryRole = body.Roles[0]
		} else {
			primaryRole = "end user"
		}
	}

	// Collect all roles
	allRoles := body.Roles
	if len(allRoles) == 0 {
		allRoles = []string{primaryRole}
	}

	// Validate all roles against known set
	validRoles := map[string]bool{
		"end user":        true,
		"content creator": true,
		"admin":           true,
		"approver":        true,
		"manager":         true,
		"auditor":         true,
	}
	for _, r := range allRoles {
		if !validRoles[r] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role: " + r})
			return
		}
	}

	hash := functions.MakeHash(body.Password)
	user, err := h.Queries.CreateUser(c.Request.Context(), database.CreateUserParams{
		Email:        body.Email,
		PasswordHash: hash,
		Name:         body.Name,
		Role:         primaryRole,
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "A user with that email already exists"})
		return
	}

	// Insert all roles into user_roles table
	for _, r := range allRoles {
		h.Queries.InsertUserRole(c.Request.Context(), database.InsertUserRoleParams{
			UserID: user.ID,
			Role:   r,
		})
	}

	audit.Log(h.Queries, c, "user_created", map[string]any{
		"created_user_id": user.ID,
		"email":           body.Email,
		"name":            body.Name,
		"roles":           allRoles,
	})

	c.JSON(http.StatusCreated, gin.H{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
		"role":  user.Role,
		"roles": allRoles,
	})
}

func (h *Handler) UpdateUserRoles(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Verify user exists
	user, err := h.Queries.GetUserByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var body struct {
		Roles []string `json:"roles" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate all roles
	validRoles := map[string]bool{
		"end user":        true,
		"content creator": true,
		"admin":           true,
		"approver":        true,
		"manager":         true,
		"auditor":         true,
	}
	for _, r := range body.Roles {
		if !validRoles[r] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role: " + r})
			return
		}
	}

	if len(body.Roles) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one role is required"})
		return
	}

	// Delete all existing roles and re-insert
	h.Queries.DeleteUserRoles(c.Request.Context(), id)

	for _, r := range body.Roles {
		h.Queries.InsertUserRole(c.Request.Context(), database.InsertUserRoleParams{
			UserID: id,
			Role:   r,
		})
	}

	// Update the primary role column to the first role
	h.Queries.UpdateUserPrimaryRole(c.Request.Context(), database.UpdateUserPrimaryRoleParams{
		ID:   id,
		Role: body.Roles[0],
	})

	audit.Log(h.Queries, c, "user_roles_updated", map[string]any{
		"target_user_id": id,
		"email":          user.Email,
		"name":           user.Name,
		"new_roles":      body.Roles,
	})

	c.JSON(http.StatusOK, gin.H{
		"id":    id,
		"roles": body.Roles,
	})
}

func (h *Handler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get user info before deleting for audit
	user, err := h.Queries.GetUserByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Delete from user_roles first (though CASCADE should handle this)
	h.Queries.DeleteUserRoles(c.Request.Context(), id)

	if err := h.Queries.DeleteUser(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	audit.Log(h.Queries, c, "user_deleted", map[string]any{
		"deleted_user_id": id,
		"email":           user.Email,
		"name":            user.Name,
	})

	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

// getUserRoles fetches all roles for a user from the user_roles junction table
func (h *Handler) getUserRoles(c *gin.Context, userID int64) []string {
	roles, err := h.Queries.GetUserRoles(c.Request.Context(), userID)
	if err != nil || len(roles) == 0 {
		return []string{}
	}
	return roles
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/admin/users", h.CreateUser)
	r.PUT("/admin/users/:id/roles", h.UpdateUserRoles)
	r.DELETE("/admin/users/:id", h.DeleteUser)
}

// RegisterListRoute exposes user listing to admin+manager (read-only).
func (h *Handler) RegisterListRoute(r *gin.RouterGroup) {
	r.GET("/admin/users", h.ListUsers)
}

// AdminEnrollUser lets an admin or manager assign a course to a user.
func (h *Handler) AdminEnrollUser(c *gin.Context) {
	var body struct {
		UserID   int64 `json:"user_id" binding:"required"`
		CourseID int64 `json:"course_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify user exists
	targetUser, err := h.Queries.GetUserByID(c.Request.Context(), body.UserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Verify course exists
	course, err := h.Queries.GetCourseByID(c.Request.Context(), body.CourseID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	// Enroll the user (no-op if already enrolled)
	err = h.Queries.EnrollInCourse(c.Request.Context(),
		database.EnrollInCourseParams{UserID: body.UserID, CourseID: body.CourseID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enroll user"})
		return
	}

	audit.Log(h.Queries, c, "user_enrolled_in_course", map[string]any{
		"target_user_id":    body.UserID,
		"target_user_name":  targetUser.Name,
		"target_user_email": targetUser.Email,
		"course_id":         body.CourseID,
		"course_title":      course.Title,
	})

	c.JSON(http.StatusOK, gin.H{
		"message":      "User enrolled",
		"user_id":      body.UserID,
		"user_name":    targetUser.Name,
		"course_id":    body.CourseID,
		"course_title": course.Title,
	})
}

// RegisterEnrollRoutes registers the enrollment endpoint under a group
// that allows both admin and manager roles.
func (h *Handler) RegisterEnrollRoutes(r *gin.RouterGroup) {
	r.POST("/admin/enrollments", h.AdminEnrollUser)
}
