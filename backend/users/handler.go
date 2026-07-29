package users

import (
	"encoding/csv"
	"fgb-lp/audit"
	database "fgb-lp/database/queries"
	"fgb-lp/functions"
	"fgb-lp/mailer"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	Pool    *pgxpool.Pool
	Queries *database.Queries
	Mailer  mailer.Sender
}

func NewHandler(pool *pgxpool.Pool, queries *database.Queries) *Handler {
	return &Handler{
		Pool:    pool,
		Queries: queries,
	}
}

// WithMailer sets the email sender (fluent builder pattern).
func (h *Handler) WithMailer(m mailer.Sender) *Handler {
	h.Mailer = m
	return h
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

	// Begin transaction for atomic role update
	tx, err := h.Pool.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback(c.Request.Context())
	txQueries := h.Queries.WithTx(tx)

	// Delete all existing roles
	txQueries.DeleteUserRoles(c.Request.Context(), id)

	// Re-insert roles
	for _, r := range body.Roles {
		txQueries.InsertUserRole(c.Request.Context(), database.InsertUserRoleParams{
			UserID: id,
			Role:   r,
		})
	}

	// Update the primary role column to the first role
	txQueries.UpdateUserPrimaryRole(c.Request.Context(), database.UpdateUserPrimaryRoleParams{
		ID:   id,
		Role: body.Roles[0],
	})

	// Commit transaction
	if err := tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit role changes"})
		return
	}

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

// validRoles is the canonical set of accepted role identifiers.
var validRoles = map[string]bool{
	"end user":        true,
	"content creator": true,
	"admin":           true,
	"approver":        true,
	"manager":         true,
	"auditor":         true,
}

// userRow is a uniform type used by both JSON and CSV paths.
// Department is optional: empty means no department assignment.
type userRow struct {
	Name       string
	Email      string
	Roles      []string
	Department string
}

// BulkCreateUsers creates multiple users via a JSON payload with
// auto-generated passwords + welcome emails.
// Each user may optionally specify a `department` (name) to be assigned to.
func (h *Handler) BulkCreateUsers(c *gin.Context) {
	type jsonInput struct {
		Name       string   `json:"name" binding:"required"`
		Email      string   `json:"email" binding:"required,email"`
		Roles      []string `json:"roles"`
		Department string   `json:"department"`
	}

	var body struct {
		Users []jsonInput `json:"users" binding:"required,min=1,max=200"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rows := make([]userRow, 0, len(body.Users))
	for _, u := range body.Users {
		roles := u.Roles
		if len(roles) == 0 {
			roles = []string{"end user"}
		}
		rows = append(rows, userRow{
			Name:       u.Name,
			Email:      u.Email,
			Roles:      roles,
			Department: strings.TrimSpace(u.Department),
		})
	}

	h.createUsersFromRows(c, rows)
}

// BulkCreateUsersCSV parses a CSV file upload and creates users.
// Expects headers: Name,Email,Roles,Department
//   - Roles is optional (semicolon-separated); defaults to "end user"
//   - Department is optional; matched case-insensitively by name and
//     auto-created if it does not yet exist.
//
// The form field name is "file". Maximum 200 rows.
func (h *Handler) BulkCreateUsersCSV(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV file is required (form field: file)"})
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	// Read header row
	headers, err := reader.Read()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read CSV header: " + err.Error()})
		return
	}

	// Normalize headers to lowercase, strip whitespace
	colIndex := map[string]int{}
	for i, h := range headers {
		colIndex[strings.ToLower(strings.TrimSpace(h))] = i
	}

	nameCol, hasName := colIndex["name"]
	emailCol, hasEmail := colIndex["email"]
	rolesCol, _ := colIndex["roles"]
	deptCol, _ := colIndex["department"]

	if !hasName || !hasEmail {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV must have 'Name' and 'Email' columns"})
		return
	}

	var rows []userRow
	line := 1 // line 1 is the header

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		line++
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Failed to read CSV line " + strconv.Itoa(line) + ": " + err.Error(),
			})
			return
		}

		if len(record) <= nameCol || len(record) <= emailCol {
			continue // skip malformed rows silently; validation catches them
		}

		name := strings.TrimSpace(record[nameCol])
		email := strings.TrimSpace(record[emailCol])

		if name == "" || email == "" {
			continue
		}

		roles := []string{"end user"}
		if rolesCol >= 0 && rolesCol < len(record) {
			raw := strings.TrimSpace(record[rolesCol])
			if raw != "" {
				parts := strings.Split(raw, ";")
				parsed := make([]string, 0, len(parts))
				for _, p := range parts {
					r := strings.TrimSpace(strings.ToLower(p))
					if r != "" && validRoles[r] {
						parsed = append(parsed, r)
					}
				}
				if len(parsed) > 0 {
					roles = parsed
				}
			}
		}

		department := ""
		if deptCol >= 0 && deptCol < len(record) {
			department = strings.TrimSpace(record[deptCol])
		}

		rows = append(rows, userRow{
			Name:       name,
			Email:      email,
			Roles:      roles,
			Department: department,
		})
	}

	if len(rows) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No valid user rows found in CSV"})
		return
	}
	if len(rows) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 200 users per upload (got " + strconv.Itoa(len(rows)) + ")"})
		return
	}

	h.createUsersFromRows(c, rows)
}

// createUsersFromRows is the shared creation pipeline: validate, create in
// a transaction, send welcome emails, audit, respond.
func (h *Handler) createUsersFromRows(c *gin.Context, rows []userRow) {
	// Pre-validate
	type rowError struct {
		Index   int    `json:"index"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Message string `json:"message"`
	}
	var validationErrors []rowError

	for i, r := range rows {
		for _, role := range r.Roles {
			if !validRoles[role] {
				validationErrors = append(validationErrors, rowError{
					Index: i, Email: r.Email, Name: r.Name,
					Message: "Invalid role: " + role,
				})
			}
		}
	}

	if len(validationErrors) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": validationErrors,
		})
		return
	}

	// Transaction
	tx, err := h.Pool.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback(c.Request.Context())
	txQueries := h.Queries.WithTx(tx)

	// Cache department name -> id so we only resolve each name once.
	deptCache := map[string]int64{}

	type createdUser struct {
		ID         int64    `json:"id"`
		Name       string   `json:"name"`
		Email      string   `json:"email"`
		Password   string   `json:"-"`
		Roles      []string `json:"roles"`
		Department string   `json:"department"`
	}

	created := make([]createdUser, 0, len(rows))

	for _, r := range rows {
		password := functions.MakeTokens()
		hash := functions.MakeHash(password)

		user, err := txQueries.CreateUser(c.Request.Context(), database.CreateUserParams{
			Email:        r.Email,
			PasswordHash: hash,
			Name:         r.Name,
			Role:         r.Roles[0],
		})
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "Failed to create user",
				"details": gin.H{"email": r.Email, "name": r.Name},
			})
			return
		}

		for _, role := range r.Roles {
			_ = txQueries.InsertUserRole(c.Request.Context(), database.InsertUserRoleParams{
				UserID: user.ID,
				Role:   role,
			})
		}

		deptName := ""
		if r.Department != "" {
			deptID, ok := deptCache[r.Department]
			if !ok {
				dept, dErr := txQueries.GetDepartmentByName(c.Request.Context(), r.Department)
				if dErr == nil {
					deptID = dept.ID
				} else {
					// Auto-create the department so bulk uploads don't require
					// pre-provisioning of every department name.
					newDept, cErr := txQueries.CreateDepartment(c.Request.Context(), r.Department)
					if cErr != nil {
						c.JSON(http.StatusInternalServerError, gin.H{
							"error":   "Failed to resolve or create department",
							"details": gin.H{"department": r.Department, "email": r.Email},
						})
						return
					}
					deptID = newDept.ID
				}
				deptCache[r.Department] = deptID
			}

			_ = txQueries.AddUserToDepartment(c.Request.Context(), database.AddUserToDepartmentParams{
				UserID:       user.ID,
				DepartmentID: deptID,
			})
			deptName = r.Department
		}

		created = append(created, createdUser{
			ID:         user.ID,
			Name:       user.Name,
			Email:      user.Email,
			Password:   password,
			Roles:      r.Roles,
			Department: deptName,
		})
	}

	if err := tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit bulk creation"})
		return
	}

	// Send welcome emails synchronously so we can report per-user status.
	// Resend API is fast enough for this to be acceptable (< 200ms per email).
	//
	// Graceful failure: if the email cannot be delivered, the plaintext
	// password would otherwise be lost (only its hash is stored). In that
	// case we return it to the admin so the credential is recoverable and
	// the user is never created with an unknown password. When the email
	// succeeds we keep the password private (omitted from the response).
	type userResponse struct {
		ID         int64    `json:"id"`
		Name       string   `json:"name"`
		Email      string   `json:"email"`
		Roles      []string `json:"roles"`
		Department string   `json:"department"`
		EmailSent  bool     `json:"email_sent"`
		// Password is only populated when EmailSent is false.
		Password string `json:"password,omitempty"`
	}

	responseUsers := make([]userResponse, len(created))
	emailSent := 0
	emailFailed := 0

	for i, cu := range created {
		responseUsers[i] = userResponse{
			ID:         cu.ID,
			Name:       cu.Name,
			Email:      cu.Email,
			Roles:      cu.Roles,
			Department: cu.Department,
		}

		if h.Mailer != nil {
			err := h.Mailer.SendWelcome(c.Request.Context(), cu.Email, cu.Name, cu.Password)
			if err != nil {
				log.Printf("[users] failed to send welcome email to %s: %v", cu.Email, err)
				emailFailed++
				responseUsers[i].EmailSent = false
				responseUsers[i].Password = cu.Password
			} else {
				emailSent++
				responseUsers[i].EmailSent = true
			}
		} else {
			// No mailer configured — the password is only available via the
			// backend logs (noopSender logs it). Return it here so the admin
			// can still hand credentials out-of-band.
			emailFailed++
			responseUsers[i].EmailSent = false
			responseUsers[i].Password = cu.Password
		}
	}

	audit.Log(h.Queries, c, "users_bulk_created", map[string]any{
		"created_count": len(created),
		"emails_sent":   emailSent,
		"emails_failed": emailFailed,
	})

	c.JSON(http.StatusCreated, gin.H{
		"created_count": len(created),
		"emails_sent":   emailSent,
		"emails_failed": emailFailed,
		"users":         responseUsers,
	})
}

// BulkCreateUsersCSVTemplate returns a downloadable CSV template that admins
// can fill in and re-upload via /admin/users/bulk/csv. The Department column
// is optional and is matched case-insensitively by name on upload
// (auto-created if it does not yet exist).
func (h *Handler) BulkCreateUsersCSVTemplate(c *gin.Context) {
	// Include existing department names so admins can copy/paste exact names.
	deps, _ := h.Queries.GetDepartments(c.Request.Context())
	depNames := make([]string, 0, len(deps))
	for _, d := range deps {
		depNames = append(depNames, d.Name)
	}

	var buf strings.Builder
	buf.WriteString("Name,Email,Roles,Department\n")
	buf.WriteString("Jane Doe,jane@example.com,end user,Operations\n")
	buf.WriteString("John Smith,john@example.com,manager;approver,Finance\n")
	buf.WriteString("Alex Lee,alex@example.com,end user,\n")
	buf.WriteString("\n")
	buf.WriteString("# Roles (semicolon-separated, optional): end user, content creator, approver, admin, manager, auditor\n")
	if len(depNames) > 0 {
		buf.WriteString("# Existing departments: " + strings.Join(depNames, "; ") + "\n")
	} else {
		buf.WriteString("# Department is optional. New names are auto-created on upload.\n")
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="bulk-users-template.csv"`)
	c.Status(http.StatusOK)
	_, _ = c.Writer.WriteString(buf.String())
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
	r.POST("/admin/users/bulk", h.BulkCreateUsers)
	r.POST("/admin/users/bulk/csv", h.BulkCreateUsersCSV)
	r.GET("/admin/users/bulk/template", h.BulkCreateUsersCSVTemplate)
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
