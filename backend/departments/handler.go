package departments

import (
	"fgb-lp/audit"
	database "fgb-lp/database/queries"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Queries *database.Queries
}

func NewHandler(queries *database.Queries) *Handler {
	return &Handler{Queries: queries}
}

// List returns all departments.
func (h *Handler) List(c *gin.Context) {
	deps, err := h.Queries.GetDepartments(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch departments"})
		return
	}
	if deps == nil {
		deps = []database.Department{}
	}
	c.JSON(http.StatusOK, deps)
}

// Create adds a new department.
func (h *Handler) Create(c *gin.Context) {
	var body struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dep, err := h.Queries.CreateDepartment(c.Request.Context(), body.Name)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "A department with that name already exists"})
		return
	}

	audit.Log(h.Queries, c, "department_created", map[string]any{
		"department_id":   dep.ID,
		"department_name": dep.Name,
	})

	c.JSON(http.StatusCreated, dep)
}

// Delete removes a department (cascades to user_departments).
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	dep, depErr := h.Queries.GetDepartmentByID(c.Request.Context(), id)
	if depErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Department not found"})
		return
	}

	if err := h.Queries.DeleteDepartment(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete department"})
		return
	}

	audit.Log(h.Queries, c, "department_deleted", map[string]any{
		"department_id":   id,
		"department_name": dep.Name,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Department deleted"})
}

// AddUser assigns a user to a department.
func (h *Handler) AddUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department ID"})
		return
	}

	var body struct {
		UserID int64 `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.Queries.AddUserToDepartment(c.Request.Context(),
		database.AddUserToDepartmentParams{
			UserID:       body.UserID,
			DepartmentID: id,
		}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add user to department"})
		return
	}

	dep, _ := h.Queries.GetDepartmentByID(c.Request.Context(), id)
	user, _ := h.Queries.GetUserByID(c.Request.Context(), body.UserID)
	depName := "department"
	userName := "user"
	if dep != (database.Department{}) {
		depName = dep.Name
	}
	if user != (database.User{}) {
		userName = user.Name
	}

	audit.Log(h.Queries, c, "user_added_to_department", map[string]any{
		"target_user_id":   body.UserID,
		"target_user_name": userName,
		"department_id":    id,
		"department_name":  depName,
	})

	c.JSON(http.StatusOK, gin.H{"message": "User added to department"})
}

// RemoveUser removes a user from a department.
func (h *Handler) RemoveUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department ID"})
		return
	}

	var body struct {
		UserID int64 `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.Queries.RemoveUserFromDepartment(c.Request.Context(),
		database.RemoveUserFromDepartmentParams{
			UserID:       body.UserID,
			DepartmentID: id,
		}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove user from department"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User removed from department"})
}

// UsersByDepartment lists all users in a department.
func (h *Handler) UsersByDepartment(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department ID"})
		return
	}

	users, err := h.Queries.GetUsersByDepartment(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	if users == nil {
		users = []database.GetUsersByDepartmentRow{}
	}
	c.JSON(http.StatusOK, users)
}

// BulkEnroll enrolls every user in a department into a course.
func (h *Handler) BulkEnroll(c *gin.Context) {
	var body struct {
		DepartmentID int64 `json:"department_id" binding:"required"`
		CourseID     int64 `json:"course_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	course, err := h.Queries.GetCourseByID(c.Request.Context(), body.CourseID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	dep := database.Department{}
	d, depErr := h.Queries.GetDepartmentByID(c.Request.Context(), body.DepartmentID)
	if depErr == nil {
		dep = d
	}

	users, err := h.Queries.GetUsersByDepartment(c.Request.Context(), body.DepartmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch department users"})
		return
	}

	enrolled := 0
	for _, u := range users {
		err := h.Queries.EnrollInCourse(c.Request.Context(),
			database.EnrollInCourseParams{UserID: u.ID, CourseID: body.CourseID})
		if err == nil {
			enrolled++
		}
	}

	audit.Log(h.Queries, c, "department_bulk_enrolled", map[string]any{
		"department_id":   body.DepartmentID,
		"department_name": dep.Name,
		"course_id":       body.CourseID,
		"course_title":    course.Title,
		"enrolled_count":  enrolled,
	})

	c.JSON(http.StatusOK, gin.H{
		"message":        "Bulk enrollment complete",
		"enrolled_count": enrolled,
		"total_in_dept":  len(users),
		"course_title":   course.Title,
	})
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/admin/departments", h.List)
	r.POST("/admin/departments", h.Create)
	r.DELETE("/admin/departments/:id", h.Delete)
	r.POST("/admin/departments/:id/users", h.AddUser)
	r.DELETE("/admin/departments/:id/users", h.RemoveUser)
	r.GET("/admin/departments/:id/users", h.UsersByDepartment)
	r.POST("/admin/departments/bulk-enroll", h.BulkEnroll)
}
