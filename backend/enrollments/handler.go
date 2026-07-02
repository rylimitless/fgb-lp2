package enrollments

import (
	"context"
	"encoding/json"
	"fgb-lp/audit"
	database "fgb-lp/database/queries"
	"fgb-lp/mailer"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	Queries *database.Queries
	Mailer  mailer.Sender
}

func NewHandler(queries *database.Queries) *Handler {
	return &Handler{Queries: queries}
}

func (h *Handler) WithMailer(m mailer.Sender) *Handler {
	h.Mailer = m
	return h
}

// ListMyEnrollments returns all enrollments for the authenticated user
// with course title, description, and status included.
func (h *Handler) ListMyEnrollments(c *gin.Context) {
	userID := c.GetInt64("user_id")

	enrollments, err := h.Queries.GetUserEnrollments(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch enrollments"})
		return
	}
	if enrollments == nil {
		enrollments = []database.GetUserEnrollmentsRow{}
	}

	// Build response with course info + progress + settings
	type EnrollmentResponse struct {
		ID                int64           `json:"id"`
		UserID            int64           `json:"user_id"`
		CourseID          int64           `json:"course_id"`
		Status            string          `json:"status"`
		ProgressPct       float64         `json:"progress_pct"`
		EnrolledAt        string          `json:"enrolled_at"`
		CompletedAt       *string         `json:"completed_at"`
		DroppedAt         *string         `json:"dropped_at"`
		CourseTitle       string          `json:"course_title"`
		CourseDescription string          `json:"course_description"`
		CourseStatus      string          `json:"course_status"`
		Settings          json.RawMessage `json:"settings"`
	}

	out := make([]EnrollmentResponse, len(enrollments))
	for i, e := range enrollments {
		var pct float64
		if e.ProgressPct.Valid {
			_ = e.ProgressPct.Scan(&pct)
		}

		var completedAt *string
		if e.CompletedAt.Valid {
			s := e.CompletedAt.Time.Format("2006-01-02 15:04")
			completedAt = &s
		}

		var droppedAt *string
		if e.DroppedAt.Valid {
			s := e.DroppedAt.Time.Format("2006-01-02 15:04")
			droppedAt = &s
		}

		// Fetch course to get settings
		course, _ := h.Queries.GetCourseByID(c.Request.Context(), e.CourseID)
		var settings json.RawMessage
		if course.Settings != nil {
			settings = json.RawMessage(course.Settings)
		}

		out[i] = EnrollmentResponse{
			ID:                e.ID,
			UserID:            e.UserID,
			CourseID:          e.CourseID,
			Status:            e.Status,
			ProgressPct:       pct,
			EnrolledAt:        e.EnrolledAt.Time.Format("2006-01-02 15:04"),
			CompletedAt:       completedAt,
			DroppedAt:         droppedAt,
			CourseTitle:       e.CourseTitle,
			CourseDescription: e.CourseDescription,
			CourseStatus:      e.CourseStatus,
			Settings:          settings,
		}
	}

	c.JSON(http.StatusOK, out)
}

// EnrollSelf lets the authenticated user enroll in a published course.
// Validates course exists, is published, and capacity is not exceeded.
// Blocks re-enrollment if previous enrollment expired without completion.
func (h *Handler) EnrollSelf(c *gin.Context) {
	userID := c.GetInt64("user_id")

	var body struct {
		CourseID int64 `json:"course_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify course exists and is published
	course, err := h.Queries.GetCourseByID(c.Request.Context(), body.CourseID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	if course.Status != "published" || !course.Approved.Bool {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Course is not available for enrollment"})
		return
	}

	// Check capacity
	if course.Capacity.Valid && course.Capacity.Int32 > 0 {
		activeCount, _ := h.Queries.CountActiveEnrollments(c.Request.Context(), body.CourseID)
		if activeCount >= int64(course.Capacity.Int32) {
			c.JSON(http.StatusConflict, gin.H{"error": "Course is at maximum capacity"})
			return
		}
	}

	// Parse course settings to check for days_to_complete expiry
	var courseSettings map[string]interface{}
	if len(course.Settings) > 0 {
		json.Unmarshal(course.Settings, &courseSettings)
	}

	// If course has a deadline, check if the user's previous enrollment expired
	if dtc, ok := courseSettings["days_to_complete"].(float64); ok && dtc > 0 {
		prevEnrollment, _ := h.Queries.GetCourseEnrollment(c.Request.Context(),
			database.GetCourseEnrollmentParams{UserID: userID, CourseID: body.CourseID})
		if prevEnrollment.EnrolledAt.Valid {
			deadline := prevEnrollment.EnrolledAt.Time.Add(time.Duration(int(dtc)) * 24 * time.Hour)
			if time.Now().After(deadline) {
				// Check if the course was actually completed
				progress, _ := h.Queries.GetLessonProgress(c.Request.Context(),
					database.GetLessonProgressParams{UserID: userID, CourseID: body.CourseID})
				if !progress.Completed {
					c.JSON(http.StatusForbidden, gin.H{
						"error":   "Enrollment deadline has passed",
						"message": fmt.Sprintf("You did not complete this course within the %.0f-day deadline. You cannot re-enroll.", dtc),
					})
					return
				}
			}
		}
	}

	// Create enrollment (or re-activate if previously dropped)
	enrollment, err := h.Queries.CreateEnrollment(c.Request.Context(),
		database.CreateEnrollmentParams{UserID: userID, CourseID: body.CourseID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enroll"})
		return
	}

	// Also create lesson_progress row for backward compatibility
	_ = h.Queries.EnrollInCourse(c.Request.Context(),
		database.EnrollInCourseParams{UserID: userID, CourseID: body.CourseID})

	audit.Log(h.Queries, c, "enrolled_in_course", map[string]any{
		"course_id":    body.CourseID,
		"course_title": course.Title,
	})

	// Send enrollment confirmation email (fire-and-forget)
	if h.Mailer != nil {
		user, err := h.Queries.GetUserByID(c.Request.Context(), userID)
		if err == nil {
			go func() {
				if err := h.Mailer.SendEnrollmentConfirmation(context.Background(), user.Email, user.Name, course.Title, course.ID); err != nil {
					log.Printf("[enrollments] failed to send enrollment email to %s: %v", user.Email, err)
					audit.Logf(h.Queries, nil, "email_enrollment_failed", "to=%s course=%q error=%v", user.Email, course.Title, err)
				} else {
					audit.Logf(h.Queries, nil, "email_enrollment_sent", "to=%s course=%q course_id=%d", user.Email, course.Title, course.ID)
				}
			}()
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":          enrollment.ID,
		"user_id":     enrollment.UserID,
		"course_id":   enrollment.CourseID,
		"status":      enrollment.Status,
		"enrolled_at": enrollment.EnrolledAt.Time.Format("2006-01-02 15:04"),
	})
}

// DropEnrollment lets a user drop an active enrollment.
func (h *Handler) DropEnrollment(c *gin.Context) {
	userID := c.GetInt64("user_id")

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid enrollment ID"})
		return
	}

	// Verify the enrollment belongs to the user
	existing, err := h.Queries.GetEnrollmentByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Enrollment not found"})
		return
	}
	if existing.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not your enrollment"})
		return
	}

	if existing.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only active enrollments can be dropped"})
		return
	}

	updated, err := h.Queries.UpdateEnrollmentStatus(c.Request.Context(),
		database.UpdateEnrollmentStatusParams{ID: id, Status: "dropped"})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to drop enrollment"})
		return
	}

	audit.Log(h.Queries, c, "dropped_enrollment", map[string]any{
		"enrollment_id": id,
		"course_id":     existing.CourseID,
		"course_title":  existing.CourseTitle,
	})

	c.JSON(http.StatusOK, gin.H{
		"id":      updated.ID,
		"status":  updated.Status,
		"message": "Enrollment dropped",
	})
}

// GetCourseEnrollmentDetails (admin) lists all enrollments for a course.
func (h *Handler) GetCourseEnrollmentDetails(c *gin.Context) {
	courseIDStr := c.Param("course_id")
	courseID, err := strconv.ParseInt(courseIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID"})
		return
	}

	// Verify course exists
	_, err = h.Queries.GetCourseByID(c.Request.Context(), courseID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	enrollments, err := h.Queries.GetCourseEnrollments(c.Request.Context(), courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch enrollments"})
		return
	}
	if enrollments == nil {
		enrollments = []database.GetCourseEnrollmentsRow{}
	}

	type EnrollmentRow struct {
		ID          int64   `json:"id"`
		UserID      int64   `json:"user_id"`
		UserName    string  `json:"user_name"`
		UserEmail   string  `json:"user_email"`
		Status      string  `json:"status"`
		ProgressPct float64 `json:"progress_pct"`
		EnrolledAt  string  `json:"enrolled_at"`
		CompletedAt *string `json:"completed_at"`
		DroppedAt   *string `json:"dropped_at"`
	}

	var activeCount, completedCount, droppedCount int64
	out := make([]EnrollmentRow, len(enrollments))
	for i, e := range enrollments {
		var pct float64
		if e.ProgressPct.Valid {
			_ = e.ProgressPct.Scan(&pct)
		}

		var completedAt *string
		if e.CompletedAt.Valid {
			s := e.CompletedAt.Time.Format("2006-01-02 15:04")
			completedAt = &s
		}

		var droppedAt *string
		if e.DroppedAt.Valid {
			s := e.DroppedAt.Time.Format("2006-01-02 15:04")
			droppedAt = &s
		}

		out[i] = EnrollmentRow{
			ID:          e.ID,
			UserID:      e.UserID,
			UserName:    e.UserName,
			UserEmail:   e.UserEmail,
			Status:      e.Status,
			ProgressPct: pct,
			EnrolledAt:  e.EnrolledAt.Time.Format("2006-01-02 15:04"),
			CompletedAt: completedAt,
			DroppedAt:   droppedAt,
		}

		switch e.Status {
		case "active", "pending":
			activeCount++
		case "completed":
			completedCount++
		case "dropped":
			droppedCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"enrollments":     out,
		"active_count":    activeCount,
		"completed_count": completedCount,
		"dropped_count":   droppedCount,
		"total_count":     len(enrollments),
	})
}

// AdminEnrollUser lets an admin/manager enroll a user in a course.
// Overrides the simpler version in users/handler.go with full enrollment lifecycle support.
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

	// Create enrollment (or re-activate)
	enrollment, err := h.Queries.CreateEnrollment(c.Request.Context(),
		database.CreateEnrollmentParams{UserID: body.UserID, CourseID: body.CourseID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enroll user"})
		return
	}

	// Also ensure lesson_progress row for backward compatibility
	_ = h.Queries.EnrollInCourse(c.Request.Context(),
		database.EnrollInCourseParams{UserID: body.UserID, CourseID: body.CourseID})

	audit.Log(h.Queries, c, "admin_enrolled_user", map[string]any{
		"target_user_id":    body.UserID,
		"target_user_name":  targetUser.Name,
		"target_user_email": targetUser.Email,
		"course_id":         body.CourseID,
		"course_title":      course.Title,
	})

	// Send enrollment confirmation email to the enrolled user (fire-and-forget)
	if h.Mailer != nil {
		go func() {
			if err := h.Mailer.SendEnrollmentConfirmation(context.Background(), targetUser.Email, targetUser.Name, course.Title, course.ID); err != nil {
				log.Printf("[enrollments] failed to send enrollment email to %s: %v", targetUser.Email, err)
				audit.Logf(h.Queries, nil, "email_enrollment_failed", "to=%s course=%q error=%v", targetUser.Email, course.Title, err)
			} else {
				audit.Logf(h.Queries, nil, "email_enrollment_sent", "to=%s course=%q course_id=%d", targetUser.Email, course.Title, course.ID)
			}
		}()
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          enrollment.ID,
		"user_id":     enrollment.UserID,
		"course_id":   enrollment.CourseID,
		"status":      enrollment.Status,
		"enrolled_at": enrollment.EnrolledAt.Time.Format("2006-01-02 15:04"),
	})
}

// BulkEnrollUsers lets an admin enroll many users in a course at once.
// Accepts { "user_ids": [1, 2, 3], "course_id": 7 }. Tracks enrolled, skipped,
// and errored users. Skips users that don't exist or are already enrolled.
func (h *Handler) BulkEnrollUsers(c *gin.Context) {
	var body struct {
		UserIDs  []int64 `json:"user_ids" binding:"required"`
		CourseID int64   `json:"course_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify course exists
	course, err := h.Queries.GetCourseByID(c.Request.Context(), body.CourseID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	var enrolledCount, skippedCount, errorCount int
	var errorDetails []map[string]any

	for _, uid := range body.UserIDs {
		// Verify user exists
		_, err := h.Queries.GetUserByID(c.Request.Context(), uid)
		if err != nil {
			skippedCount++
			errorDetails = append(errorDetails, map[string]any{
				"user_id": uid,
				"reason":  "User not found",
			})
			continue
		}

		// Create enrollment
		_, err = h.Queries.CreateEnrollment(c.Request.Context(),
			database.CreateEnrollmentParams{UserID: uid, CourseID: body.CourseID})
		if err != nil {
			errStr := strings.ToLower(err.Error())
			// Check for duplicate / already-enrolled
			if strings.Contains(errStr, "duplicate") || strings.Contains(errStr, "unique") || strings.Contains(errStr, "already") {
				skippedCount++
				errorDetails = append(errorDetails, map[string]any{
					"user_id": uid,
					"reason":  "Already enrolled",
				})
				// Still ensure lesson_progress exists
				_ = h.Queries.EnrollInCourse(c.Request.Context(),
					database.EnrollInCourseParams{UserID: uid, CourseID: body.CourseID})
				continue
			}

			errorCount++
			errorDetails = append(errorDetails, map[string]any{
				"user_id": uid,
				"reason":  fmt.Sprintf("CreateEnrollment failed: %s", errStr),
			})
			continue
		}

		// Ensure lesson_progress row for backward compatibility
		_ = h.Queries.EnrollInCourse(c.Request.Context(),
			database.EnrollInCourseParams{UserID: uid, CourseID: body.CourseID})

		enrolledCount++
	}

	// Write a single audit log entry
	audit.Log(h.Queries, c, "bulk_enrollment", map[string]any{
		"course_id":       body.CourseID,
		"course_title":    course.Title,
		"enrolled_count":  enrolledCount,
		"skipped_count":   skippedCount,
		"error_count":     errorCount,
		"total_requested": len(body.UserIDs),
	})

	c.JSON(http.StatusOK, gin.H{
		"enrolled_count": enrolledCount,
		"skipped_count":  skippedCount,
		"error_count":    errorCount,
		"course_title":   course.Title,
		"errors":         errorDetails,
	})
}

// AdminRemoveEnrollment lets an admin/manager remove a user's enrollment.
func (h *Handler) AdminRemoveEnrollment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid enrollment ID"})
		return
	}

	// Verify enrollment exists
	existing, err := h.Queries.GetEnrollmentByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Enrollment not found"})
		return
	}

	if err := h.Queries.DeleteEnrollment(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove enrollment"})
		return
	}

	audit.Log(h.Queries, c, "admin_removed_enrollment", map[string]any{
		"enrollment_id":  id,
		"target_user_id": existing.UserID,
		"course_id":      existing.CourseID,
		"course_title":   existing.CourseTitle,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Enrollment removed"})
}

// UpdateProgress syncs enrollment progress_pct from lesson_progress on completion.
// Called by the lesson save-progress endpoint for consistency.
func (h *Handler) UpdateProgress(c *gin.Context) {
	var body struct {
		CourseID    int64  `json:"course_id"`
		ProgressPct string `json:"progress_pct"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetInt64("user_id")

	// Find the enrollment
	enrollment, err := h.Queries.GetCourseEnrollment(c.Request.Context(),
		database.GetCourseEnrollmentParams{UserID: userID, CourseID: body.CourseID})
	if err != nil {
		// No enrollment record yet – silently skip
		c.JSON(http.StatusOK, gin.H{"message": "No enrollment found"})
		return
	}

	pct := pgtype.Numeric{}
	pct.Scan(body.ProgressPct)

	_, err = h.Queries.UpdateEnrollmentProgress(c.Request.Context(),
		database.UpdateEnrollmentProgressParams{ID: enrollment.ID, ProgressPct: pct})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update progress"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Progress updated"})
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/enrollments", h.ListMyEnrollments)
	r.POST("/enrollments", h.EnrollSelf)
	r.PUT("/enrollments/:id/drop", h.DropEnrollment)
	r.POST("/enrollments/progress", h.UpdateProgress)
}

func (h *Handler) RegisterAdminRoutes(r *gin.RouterGroup) {
	r.GET("/admin/enrollments/course/:course_id", h.GetCourseEnrollmentDetails)
	r.POST("/admin/enrollments", h.AdminEnrollUser)
	r.POST("/admin/enrollments/bulk", h.BulkEnrollUsers)
	r.DELETE("/admin/enrollments/:id", h.AdminRemoveEnrollment)
}
