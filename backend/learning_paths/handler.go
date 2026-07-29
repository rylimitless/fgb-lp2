package learning_paths

import (
	"context"
	"fgb-lp/audit"
	database "fgb-lp/database/queries"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	Queries *database.Queries
	Pool    *pgxpool.Pool
}

func NewHandler(queries *database.Queries, pool *pgxpool.Pool) *Handler {
	return &Handler{Queries: queries, Pool: pool}
}

// pathProgress is computed from underlying course enrollments.
type pathProgress struct {
	TotalCourses      int     `json:"total_courses"`
	CompletedCourses  int     `json:"completed_courses"`
	RequiredTotal     int     `json:"required_total"`
	RequiredCompleted int     `json:"required_completed"`
	ProgressPct       float64 `json:"progress_pct"`
	IsComplete        bool    `json:"is_complete"`
}

// computeProgress reads all course enrollments for (user, path) and derives progress.
// Progress is "combined": completed/total across all courses (required + optional).
// Completion requires ALL required courses to be done (optional don't block).
func (h *Handler) computeProgress(ctx context.Context, userID, pathID int64) (*pathProgress, error) {
	var total, completed, reqTotal, reqCompleted int
	err := h.Pool.QueryRow(ctx, `
		select
		  count(*) as total,
		  count(*) filter (where e.status = 'completed') as completed,
		  count(*) filter (where lpc.is_required) as req_total,
		  count(*) filter (where lpc.is_required and e.status = 'completed') as req_completed
		from learning_path_courses lpc
		left join enrollments e
		  on e.course_id = lpc.course_id
		 and e.user_id = $1
		 and e.status <> 'dropped'
		where lpc.learning_path_id = $2`, userID, pathID).Scan(&total, &completed, &reqTotal, &reqCompleted)
	if err != nil {
		return nil, fmt.Errorf("compute progress: %w", err)
	}

	pct := 0.0
	if total > 0 {
		pct = float64(completed) / float64(total) * 100
	}
	// A path with no required courses falls back to "all courses done".
	complete := false
	if reqTotal > 0 {
		complete = reqCompleted == reqTotal
	} else if total > 0 {
		complete = completed == total
	}

	return &pathProgress{
		TotalCourses:      total,
		CompletedCourses:  completed,
		RequiredTotal:     reqTotal,
		RequiredCompleted: reqCompleted,
		ProgressPct:       pct,
		IsComplete:        complete,
	}, nil
}

// RecomputePathProgress is the public hook called by other packages (e.g. lessons.SaveProgress)
// after a course is marked complete. It recomputes progress for every active path enrollment
// of (userID, courseID) and issues a path certificate when the path becomes complete.
// Safe to call from a goroutine; never returns an error to the caller (logs only).
func (h *Handler) RecomputePathProgress(ctx context.Context, userID, courseID int64) {
	if h == nil || h.Pool == nil {
		return
	}

	// Find every active path enrollment that contains this course.
	rows, err := h.Pool.Query(ctx, `
		select lpe.id, lpe.learning_path_id, lpe.user_id
		from learning_path_enrollments lpe
		join learning_path_courses lpc on lpc.learning_path_id = lpe.learning_path_id
		where lpe.user_id = $1
		  and lpe.status = 'active'
		  and lpc.course_id = $2`, userID, courseID)
	if err != nil {
		log.Printf("[learning_paths] failed to find path enrollments for user=%d course=%d: %v", userID, courseID, err)
		return
	}
	type ref struct {
		EnrollmentID, PathID, UserID int64
	}
	var refs []ref
	for rows.Next() {
		var r ref
		if err := rows.Scan(&r.EnrollmentID, &r.PathID, &r.UserID); err != nil {
			rows.Close()
			log.Printf("[learning_paths] scan path enrollment: %v", err)
			return
		}
		refs = append(refs, r)
	}
	rows.Close()

	for _, r := range refs {
		if err := h.finalizePathEnrollment(ctx, r.UserID, r.PathID, r.EnrollmentID); err != nil {
			log.Printf("[learning_paths] finalize path=%d user=%d: %v", r.PathID, r.UserID, err)
		}
	}
}

// finalizePathEnrollment recomputes progress, persists it, and issues a cert on completion.
func (h *Handler) finalizePathEnrollment(ctx context.Context, userID, pathID, enrollmentID int64) error {
	prog, err := h.computeProgress(ctx, userID, pathID)
	if err != nil {
		return err
	}

	// Persist progress + flip status if complete.
	_, err = h.Pool.Exec(ctx, `
		update learning_path_enrollments
		   set progress_pct = $2,
		       status = case when $3 then 'completed' else status end,
		       completed_at = case when $3 then coalesce(completed_at, now()) else completed_at end
		 where id = $1 and status = 'active'`,
		enrollmentID, prog.ProgressPct, prog.IsComplete)
	if err != nil {
		return fmt.Errorf("persist progress: %w", err)
	}

	if prog.IsComplete {
		if _, err := h.issuePathCertificate(ctx, userID, pathID, prog.ProgressPct); err != nil {
			return fmt.Errorf("issue cert: %w", err)
		}
		audit.Logf(h.Queries, nil, "learning_path:completed",
			"user=%d path=%d progress=%.2f", userID, pathID, prog.ProgressPct)
	}
	return nil
}

// issuePathCertificate upserts a learning_path_certificates row. Tier is derived from progress.
func (h *Handler) issuePathCertificate(ctx context.Context, userID, pathID int64, scorePct float64) (created bool, err error) {
	tier := "bronze"
	if scorePct >= 90 {
		tier = "gold"
	} else if scorePct >= 75 {
		tier = "silver"
	}
	code := uuid.New().String()
	_, err = h.Pool.Exec(ctx, `
		insert into learning_path_certificates (user_id, learning_path_id, certificate_code, score_pct, tier)
		values ($1, $2, $3, $4, $5)
		on conflict (user_id, learning_path_id) do nothing`,
		userID, pathID, code, scorePct, tier)
	if err != nil {
		return false, err
	}
	return true, nil
}

// enrollUserInCourses enrolls the user into every course on the path.
//   - Skips courses the user has already completed (those count toward path progress already).
//   - Re-activates dropped enrollments.
//   - Forces re-enroll for expired-and-incomplete courses (deadline bypass for path enrollment).
//
// Returns counts for caller to log/audit.
func (h *Handler) enrollUserInCourses(ctx context.Context, userID, pathID int64) (enrolled, skipped, reactivated int, err error) {
	rows, err := h.Pool.Query(ctx, `
		select lpc.course_id,
		       coalesce(e.status, 'none') as status
		from learning_path_courses lpc
		left join enrollments e
		  on e.course_id = lpc.course_id
		 and e.user_id = $1
		where lpc.learning_path_id = $2
		order by lpc.sort_order asc`, userID, pathID)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("fetch path courses: %w", err)
	}
	type row struct {
		CourseID int64
		Status   string
	}
	var items []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.CourseID, &r.Status); err != nil {
			rows.Close()
			return 0, 0, 0, err
		}
		items = append(items, r)
	}
	rows.Close()

	for _, it := range items {
		switch it.Status {
		case "completed":
			// Already done — counts toward path progress, leave alone.
			skipped++
		case "active":
			// Already enrolled.
			skipped++
		case "dropped":
			// Re-activate.
			if _, err := h.Pool.Exec(ctx,
				`update enrollments set status='active', dropped_at=null where user_id=$1 and course_id=$2`,
				userID, it.CourseID); err != nil {
				return enrolled, skipped, reactivated, err
			}
			_ = h.Queries.EnrollInCourse(ctx, database.EnrollInCourseParams{UserID: userID, CourseID: it.CourseID})
			reactivated++
		default: // none — never enrolled, or expired-and-incomplete enrollment was hard-deleted
			// CreateEnrollment upserts to 'active' if a row exists, otherwise inserts.
			if _, err := h.Queries.CreateEnrollment(ctx,
				database.CreateEnrollmentParams{UserID: userID, CourseID: it.CourseID}); err != nil {
				return enrolled, skipped, reactivated, err
			}
			_ = h.Queries.EnrollInCourse(ctx, database.EnrollInCourseParams{UserID: userID, CourseID: it.CourseID})
			enrolled++
		}
	}
	return enrolled, skipped, reactivated, nil
}

// enrollUserInPath is the shared entry point for self-enroll and admin-enroll.
// It (1) creates/re-activates the path enrollment, (2) cascades into course enrollments,
// (3) computes initial progress (so already-completed courses count immediately).
func (h *Handler) enrollUserInPath(ctx context.Context, userID, pathID int64) (*database.LearningPathEnrollment, error) {
	// Verify path is published.
	path, err := h.Queries.GetPublishedLearningPathByID(ctx, pathID)
	if err != nil {
		return nil, fmt.Errorf("learning path not found or not published")
	}
	_ = path

	// Upsert path enrollment (re-activates dropped).
	enrollment, err := h.Queries.EnrollInLearningPath(ctx, database.EnrollInLearningPathParams{
		UserID:         userID,
		LearningPathID: pathID,
	})
	if err != nil {
		return nil, fmt.Errorf("create path enrollment: %w", err)
	}

	// Cascade into courses.
	if _, _, _, err := h.enrollUserInCourses(ctx, userID, pathID); err != nil {
		return nil, fmt.Errorf("cascade enroll: %w", err)
	}

	// Compute and persist initial progress (handles pre-completed courses).
	if err := h.finalizePathEnrollment(ctx, userID, pathID, enrollment.ID); err != nil {
		log.Printf("[learning_paths] finalize on enroll user=%d path=%d: %v", userID, pathID, err)
	}

	// Refetch to reflect any status/progress updates.
	updated, err := h.Queries.GetLearningPathEnrollment(ctx, database.GetLearningPathEnrollmentParams{
		UserID:         userID,
		LearningPathID: pathID,
	})
	if err != nil {
		return &enrollment, nil
	}
	return &updated, nil
}

// --- Admin routes ---

func (h *Handler) List(c *gin.Context) {
	paths, err := h.Queries.GetLearningPaths(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if paths == nil {
		paths = []database.LearningPath{}
	}
	c.JSON(http.StatusOK, paths)
}

// GetByID returns a single learning path by id (admin view — any status).
func (h *Handler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	path, err := h.Queries.GetLearningPathByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "learning path not found"})
		return
	}
	c.JSON(http.StatusOK, path)
}

type createBody struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
}

func (h *Handler) Create(c *gin.Context) {
	var body createBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetInt64("user_id")

	path, err := h.Queries.CreateLearningPath(c.Request.Context(), database.CreateLearningPathParams{
		Title:       body.Title,
		Description: body.Description,
		CreatedBy:   userID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	audit.Log(h.Queries, c, "learning_path:create", map[string]any{
		"path_id": path.ID,
		"title":   path.Title,
	})
	c.JSON(http.StatusCreated, path)
}

type updateBody struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var body updateBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update title/description
	if body.Title != "" || body.Description != "" {
		existing, err := h.Queries.GetLearningPathByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "learning path not found"})
			return
		}
		title := body.Title
		if title == "" {
			title = existing.Title
		}
		desc := body.Description
		if desc == "" {
			desc = existing.Description
		}
		path, err := h.Queries.UpdateLearningPath(c.Request.Context(), database.UpdateLearningPathParams{
			ID:          id,
			Title:       title,
			Description: desc,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		audit.Log(h.Queries, c, "learning_path:update", map[string]any{
			"path_id": path.ID,
			"title":   path.Title,
		})
	}

	// Update status separately if provided
	if body.Status != "" {
		path, err := h.Queries.UpdateLearningPathStatus(c.Request.Context(), database.UpdateLearningPathStatusParams{
			ID:     id,
			Status: body.Status,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		audit.Log(h.Queries, c, "learning_path:status_change", map[string]any{
			"path_id": path.ID,
			"status":  path.Status,
		})
		c.JSON(http.StatusOK, path)
		return
	}

	updated, _ := h.Queries.GetLearningPathByID(c.Request.Context(), id)
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	existing, err := h.Queries.GetLearningPathByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "learning path not found"})
		return
	}

	if err := h.Queries.DeleteLearningPath(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	audit.Log(h.Queries, c, "learning_path:delete", map[string]any{
		"path_id": existing.ID,
		"title":   existing.Title,
	})
	c.JSON(http.StatusOK, gin.H{"message": "learning path deleted"})
}

type addCourseBody struct {
	CourseID   int64 `json:"course_id" binding:"required"`
	SortOrder  int32 `json:"sort_order"`
	IsRequired *bool `json:"is_required"`
}

func (h *Handler) AddCourse(c *gin.Context) {
	pathID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var body addCourseBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	isRequired := true
	if body.IsRequired != nil {
		isRequired = *body.IsRequired
	}

	entry, err := h.Queries.AddCourseToPath(c.Request.Context(), database.AddCourseToPathParams{
		LearningPathID: pathID,
		CourseID:       body.CourseID,
		SortOrder:      body.SortOrder,
		IsRequired:     isRequired,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	audit.Log(h.Queries, c, "learning_path:add_course", map[string]any{
		"path_id":   pathID,
		"course_id": body.CourseID,
	})

	// If path is published, auto-enroll all active path members into the new course
	// and recompute their progress.
	path, perr := h.Queries.GetLearningPathByID(c.Request.Context(), pathID)
	if perr == nil && path.Status == "published" {
		go h.cascadeNewCourseToMembers(context.Background(), pathID, body.CourseID)
	}

	c.JSON(http.StatusCreated, entry)
}

// cascadeNewCourseToMembers enrolls all active members of a path into a newly-added course,
// then recomputes progress so their percent stays accurate.
func (h *Handler) cascadeNewCourseToMembers(ctx context.Context, pathID, courseID int64) {
	rows, err := h.Pool.Query(ctx,
		`select user_id from learning_path_enrollments where learning_path_id=$1 and status='active'`,
		pathID)
	if err != nil {
		log.Printf("[learning_paths] cascade: list members: %v", err)
		return
	}
	var userIDs []int64
	for rows.Next() {
		var uid int64
		if err := rows.Scan(&uid); err != nil {
			rows.Close()
			return
		}
		userIDs = append(userIDs, uid)
	}
	rows.Close()

	for _, uid := range userIDs {
		// Skip if already enrolled or completed.
		var status string
		err := h.Pool.QueryRow(ctx,
			`select coalesce((select status from enrollments where user_id=$1 and course_id=$2), 'none')`,
			uid, courseID).Scan(&status)
		if err != nil {
			log.Printf("[learning_paths] cascade: check status user=%d course=%d: %v", uid, courseID, err)
			continue
		}
		if status == "active" || status == "completed" {
			continue
		}
		if _, err := h.Queries.CreateEnrollment(ctx,
			database.CreateEnrollmentParams{UserID: uid, CourseID: courseID}); err != nil {
			log.Printf("[learning_paths] cascade: enroll user=%d course=%d: %v", uid, courseID, err)
			continue
		}
		_ = h.Queries.EnrollInCourse(ctx, database.EnrollInCourseParams{UserID: uid, CourseID: courseID})
	}
}

func (h *Handler) RemoveCourse(c *gin.Context) {
	pathID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path id"})
		return
	}
	courseID, err := strconv.ParseInt(c.Param("courseId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}

	if err := h.Queries.RemoveCourseFromPath(c.Request.Context(), database.RemoveCourseFromPathParams{
		LearningPathID: pathID,
		CourseID:       courseID,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	audit.Log(h.Queries, c, "learning_path:remove_course", map[string]any{
		"path_id":   pathID,
		"course_id": courseID,
	})

	// Recompute progress for active members (a required course removal may complete the path).
	go func() {
		ctx := context.Background()
		rows, err := h.Pool.Query(ctx,
			`select id, user_id from learning_path_enrollments where learning_path_id=$1 and status='active'`,
			pathID)
		if err != nil {
			return
		}
		type ref struct {
			ID, UserID int64
		}
		var refs []ref
		for rows.Next() {
			var r ref
			if rows.Scan(&r.ID, &r.UserID) == nil {
				refs = append(refs, r)
			}
		}
		rows.Close()
		for _, r := range refs {
			if err := h.finalizePathEnrollment(ctx, r.UserID, pathID, r.ID); err != nil {
				log.Printf("[learning_paths] recompute after remove: path=%d user=%d: %v", pathID, r.UserID, err)
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{"message": "course removed from path"})
}

type reorderBody struct {
	CourseID   int64 `json:"course_id" binding:"required"`
	SortOrder  int32 `json:"sort_order"`
	IsRequired bool  `json:"is_required"`
}

func (h *Handler) ReorderCourse(c *gin.Context) {
	pathID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var body reorderBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entry, err := h.Queries.UpdatePathCourseOrder(c.Request.Context(), database.UpdatePathCourseOrderParams{
		LearningPathID: pathID,
		CourseID:       body.CourseID,
		SortOrder:      body.SortOrder,
		IsRequired:     body.IsRequired,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, entry)
}

func (h *Handler) GetCourses(c *gin.Context) {
	pathID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	courses, err := h.Queries.GetPathCourses(c.Request.Context(), pathID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if courses == nil {
		courses = []database.GetPathCoursesRow{}
	}
	c.JSON(http.StatusOK, courses)
}

// --- Admin enrollment of users into paths ---

func (h *Handler) AdminEnrollUser(c *gin.Context) {
	pathID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body struct {
		UserID int64 `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify user exists.
	if _, err := h.Queries.GetUserByID(c.Request.Context(), body.UserID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	enrollment, err := h.enrollUserInPath(c.Request.Context(), body.UserID, pathID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	audit.Log(h.Queries, c, "learning_path:admin_enroll", map[string]any{
		"path_id": pathID,
		"user_id": body.UserID,
	})
	c.JSON(http.StatusCreated, enrollment)
}

func (h *Handler) AdminBulkEnroll(c *gin.Context) {
	pathID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body struct {
		UserIDs []int64 `json:"user_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var enrolled, skipped, errors int
	for _, uid := range body.UserIDs {
		if _, err := h.Queries.GetUserByID(c.Request.Context(), uid); err != nil {
			skipped++
			continue
		}
		if _, err := h.enrollUserInPath(c.Request.Context(), uid, pathID); err != nil {
			errors++
			continue
		}
		enrolled++
	}

	audit.Log(h.Queries, c, "learning_path:bulk_enroll", map[string]any{
		"path_id":         pathID,
		"enrolled":        enrolled,
		"skipped":         skipped,
		"errors":          errors,
		"total_requested": len(body.UserIDs),
	})

	c.JSON(http.StatusOK, gin.H{
		"enrolled_count": enrolled,
		"skipped_count":  skipped,
		"error_count":    errors,
		"total":          len(body.UserIDs),
	})
}

// AdminEnrollDepartment enrolls every user in a department into a path.
// Mirrors departments.BulkEnroll (which is course-scoped) but cascades into
// all courses on the path via enrollUserInPath. Idempotent: re-activates
// dropped enrollments and skips already-completed courses.
// POST /admin/learning-paths/:id/enroll/department  { department_id }
func (h *Handler) AdminEnrollDepartment(c *gin.Context) {
	pathID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body struct {
		DepartmentID int64 `json:"department_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dep := database.Department{}
	d, depErr := h.Queries.GetDepartmentByID(c.Request.Context(), body.DepartmentID)
	if depErr == nil {
		dep = d
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "Department not found"})
		return
	}

	users, err := h.Queries.GetUsersByDepartment(c.Request.Context(), body.DepartmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch department users"})
		return
	}

	var enrolled, errors int
	for _, u := range users {
		if _, err := h.enrollUserInPath(c.Request.Context(), u.ID, pathID); err != nil {
			errors++
			continue
		}
		enrolled++
	}

	audit.Log(h.Queries, c, "learning_path:department_enroll", map[string]any{
		"path_id":         pathID,
		"department_id":   body.DepartmentID,
		"department_name": dep.Name,
		"enrolled_count":  enrolled,
		"total_in_dept":   len(users),
	})

	c.JSON(http.StatusOK, gin.H{
		"enrolled_count": enrolled,
		"error_count":    errors,
		"total_in_dept":  len(users),
		"department":     dep.Name,
	})
}

// ListPathEnrollments (admin) returns all enrollments for a path with user info + progress.
func (h *Handler) ListPathEnrollments(c *gin.Context) {
	pathID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	rows, err := h.Pool.Query(c.Request.Context(), `
		select lpe.id, lpe.user_id, u.name, u.email, lpe.status, lpe.progress_pct,
		       lpe.started_at, lpe.completed_at, lpe.dropped_at
		from learning_path_enrollments lpe
		join users u on u.id = lpe.user_id
		where lpe.learning_path_id = $1
		order by lpe.started_at desc`, pathID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type enr struct {
		ID          int64              `json:"id"`
		UserID      int64              `json:"user_id"`
		UserName    string             `json:"user_name"`
		UserEmail   string             `json:"user_email"`
		Status      string             `json:"status"`
		ProgressPct pgtype.Numeric     `json:"progress_pct"`
		StartedAt   pgtype.Timestamptz `json:"started_at"`
		CompletedAt pgtype.Timestamptz `json:"completed_at"`
		DroppedAt   pgtype.Timestamptz `json:"dropped_at"`
	}
	out := []enr{}
	for rows.Next() {
		var e enr
		if err := rows.Scan(&e.ID, &e.UserID, &e.UserName, &e.UserEmail, &e.Status,
			&e.ProgressPct, &e.StartedAt, &e.CompletedAt, &e.DroppedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		out = append(out, e)
	}
	c.JSON(http.StatusOK, out)
}

// --- Learner routes ---

func (h *Handler) ListPublished(c *gin.Context) {
	paths, err := h.Queries.GetPublishedLearningPaths(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if paths == nil {
		paths = []database.LearningPath{}
	}
	c.JSON(http.StatusOK, paths)
}

func (h *Handler) GetPublished(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	path, err := h.Queries.GetPublishedLearningPathByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "learning path not found"})
		return
	}

	// Get courses in this path with the current user's per-course enrollment status.
	userID := c.GetInt64("user_id")
	courses, _ := h.getPathCoursesWithUserProgress(c.Request.Context(), id, userID)
	if courses == nil {
		courses = []map[string]any{}
	}

	// Get user enrollment if authenticated
	var enrollment *database.LearningPathEnrollment
	var progress *pathProgress
	if userID > 0 {
		enr, err := h.Queries.GetLearningPathEnrollment(c.Request.Context(), database.GetLearningPathEnrollmentParams{
			UserID:         userID,
			LearningPathID: id,
		})
		if err == nil {
			enrollment = &enr
		}
		progress, _ = h.computeProgress(c.Request.Context(), userID, id)
	}

	c.JSON(http.StatusOK, gin.H{
		"path":       path,
		"courses":    courses,
		"enrollment": enrollment,
		"progress":   progress,
	})
}

// getPathCoursesWithUserProgress returns each course in the path plus the user's enrollment status.
func (h *Handler) getPathCoursesWithUserProgress(ctx context.Context, pathID, userID int64) ([]map[string]any, error) {
	rows, err := h.Pool.Query(ctx, `
		select lpc.id, lpc.course_id, lpc.sort_order, lpc.is_required,
		       c.title, c.description, c.status as course_status,
		       coalesce(e.status, 'none') as user_status,
		       coalesce(e.progress_pct, 0) as user_progress
		from learning_path_courses lpc
		join courses c on c.id = lpc.course_id
		left join enrollments e
		  on e.course_id = lpc.course_id
		 and e.user_id = $1
		 and e.status <> 'dropped'
		where lpc.learning_path_id = $2
		order by lpc.sort_order asc`, userID, pathID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var id, courseID int64
		var sortOrder int32
		var isRequired bool
		var title, description, courseStatus, userStatus string
		var userProgress pgtype.Numeric
		if err := rows.Scan(&id, &courseID, &sortOrder, &isRequired, &title, &description,
			&courseStatus, &userStatus, &userProgress); err != nil {
			return nil, err
		}
		out = append(out, gin.H{
			"id":                 id,
			"learning_path_id":   pathID,
			"course_id":          courseID,
			"sort_order":         sortOrder,
			"is_required":        isRequired,
			"course_title":       title,
			"course_description": description,
			"course_status":      courseStatus,
			"user_status":        userStatus, // 'none' | 'active' | 'completed'
			"user_progress":      userProgress,
		})
	}
	return out, nil
}

func (h *Handler) EnrollSelf(c *gin.Context) {
	pathID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	userID := c.GetInt64("user_id")

	enrollment, err := h.enrollUserInPath(c.Request.Context(), userID, pathID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	audit.Log(h.Queries, c, "learning_path:enroll", map[string]any{
		"path_id": pathID,
	})
	c.JSON(http.StatusCreated, enrollment)
}

func (h *Handler) MyEnrollments(c *gin.Context) {
	userID := c.GetInt64("user_id")

	enrollments, err := h.Queries.GetUserLearningPathEnrollments(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if enrollments == nil {
		enrollments = []database.GetUserLearningPathEnrollmentsRow{}
	}
	c.JSON(http.StatusOK, enrollments)
}

// DropEnrollment — IDOR-safe: requires the enrollment to belong to the requesting user.
func (h *Handler) DropEnrollment(c *gin.Context) {
	enrID, err := strconv.ParseInt(c.Param("enrollmentId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid enrollment id"})
		return
	}
	userID := c.GetInt64("user_id")

	// Verify ownership (fixes audit item H1).
	var ownerID int64
	err = h.Pool.QueryRow(c.Request.Context(),
		`select user_id from learning_path_enrollments where id = $1`, enrID).Scan(&ownerID)
	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "enrollment not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ownerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	enrollment, err := h.Queries.DropLearningPathEnrollment(c.Request.Context(), enrID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	audit.Log(h.Queries, c, "learning_path:drop", map[string]any{
		"enrollment_id": enrollment.ID,
	})
	c.JSON(http.StatusOK, enrollment)
}

// MyPathCertificates returns learning-path certs earned by the current user.
func (h *Handler) MyPathCertificates(c *gin.Context) {
	userID := c.GetInt64("user_id")
	rows, err := h.Pool.Query(c.Request.Context(), `
		select lpc.id, lpc.user_id, lpc.learning_path_id, lpc.certificate_code,
		       lpc.score_pct, lpc.tier, lpc.issued_at, lp.title as path_title
		from learning_path_certificates lpc
		join learning_paths lp on lp.id = lpc.learning_path_id
		where lpc.user_id = $1
		order by lpc.issued_at desc`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var id, userID, pathID int64
		var code, tier, pathTitle string
		var scorePct pgtype.Numeric
		var issuedAt pgtype.Timestamptz
		if err := rows.Scan(&id, &userID, &pathID, &code, &scorePct, &tier, &issuedAt, &pathTitle); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		out = append(out, gin.H{
			"id":               id,
			"user_id":          userID,
			"learning_path_id": pathID,
			"certificate_code": code,
			"score_pct":        scorePct,
			"tier":             tier,
			"issued_at":        issuedAt,
			"path_title":       pathTitle,
		})
	}
	c.JSON(http.StatusOK, out)
}

// RegisterRoutes wires admin endpoints.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/admin/learning-paths", h.List)
	r.GET("/admin/learning-paths/:id", h.GetByID)
	r.POST("/admin/learning-paths", h.Create)
	r.PUT("/admin/learning-paths/:id", h.Update)
	r.DELETE("/admin/learning-paths/:id", h.Delete)
	r.POST("/admin/learning-paths/:id/courses", h.AddCourse)
	r.DELETE("/admin/learning-paths/:id/courses/:courseId", h.RemoveCourse)
	r.PUT("/admin/learning-paths/:id/reorder", h.ReorderCourse)
	r.GET("/admin/learning-paths/:id/courses", h.GetCourses)
	// Admin enrollment management
	r.GET("/admin/learning-paths/:id/enrollments", h.ListPathEnrollments)
	r.POST("/admin/learning-paths/:id/enroll", h.AdminEnrollUser)
	r.POST("/admin/learning-paths/:id/enroll/bulk", h.AdminBulkEnroll)
	r.POST("/admin/learning-paths/:id/enroll/department", h.AdminEnrollDepartment)
}

// RegisterLearnerRoutes wires learner-facing endpoints.
func (h *Handler) RegisterLearnerRoutes(r *gin.RouterGroup) {
	r.GET("/learning-paths", h.ListPublished)
	r.GET("/learning-paths/:id", h.GetPublished)
	r.POST("/learning-paths/:id/enroll", h.EnrollSelf)
	r.GET("/learning-paths/my-enrollments", h.MyEnrollments)
	r.GET("/learning-paths/my-certificates", h.MyPathCertificates)
	r.PUT("/learning-paths/enrollments/:enrollmentId/drop", h.DropEnrollment)
}
