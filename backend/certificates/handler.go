package certificates

import (
	"context"
	"fgb-lp/audit"
	database "fgb-lp/database/queries"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	Queries *database.Queries
	Pool    *pgxpool.Pool
}

func NewHandler(queries *database.Queries, pool *pgxpool.Pool) *Handler {
	return &Handler{Queries: queries, Pool: pool}
}

// Certificate represents a certificate row from the database.
type Certificate struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	CourseID        int64     `json:"course_id"`
	CertificateCode string    `json:"certificate_code"`
	ScorePct        float64   `json:"score_pct"`
	Tier            string    `json:"tier"`
	IssuedAt        time.Time `json:"issued_at"`
}

// CertificateWithCourse includes course title and user name for display.
type CertificateWithCourse struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	CourseID        int64     `json:"course_id"`
	CertificateCode string    `json:"certificate_code"`
	ScorePct        float64   `json:"score_pct"`
	Tier            string    `json:"tier"`
	IssuedAt        time.Time `json:"issued_at"`
	CourseTitle     string    `json:"course_title"`
	UserName        string    `json:"user_name"`
}

// ListCertificates returns all certificates for the authenticated user with course info.
// GET /certificates
func (h *Handler) ListCertificates(c *gin.Context) {
	userID := c.GetInt64("user_id")

	rows, err := h.Pool.Query(c.Request.Context(), `
		SELECT c.id, c.user_id, c.course_id, c.certificate_code, c.score_pct, c.tier, c.issued_at,
		       co.title AS course_title, u.name AS user_name
		FROM certificates c
		JOIN courses co ON co.id = c.course_id
		JOIN users u ON u.id = c.user_id
		WHERE c.user_id = $1
		ORDER BY c.issued_at DESC`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch certificates"})
		return
	}
	defer rows.Close()

	out := []CertificateWithCourse{}
	for rows.Next() {
		var cert CertificateWithCourse
		if err := rows.Scan(&cert.ID, &cert.UserID, &cert.CourseID, &cert.CertificateCode,
			&cert.ScorePct, &cert.Tier, &cert.IssuedAt, &cert.CourseTitle, &cert.UserName); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan certificate"})
			return
		}
		out = append(out, cert)
	}
	if out == nil {
		out = []CertificateWithCourse{}
	}

	c.JSON(http.StatusOK, out)
}

// GetCertificateByCode returns a certificate by its code for public verification.
// No auth required. GET /certificates/:code
func (h *Handler) GetCertificateByCode(c *gin.Context) {
	code := c.Param("code")

	var cert CertificateWithCourse
	err := h.Pool.QueryRow(c.Request.Context(), `
		SELECT c.id, c.user_id, c.course_id, c.certificate_code, c.score_pct, c.tier, c.issued_at,
		       co.title AS course_title, u.name AS user_name
		FROM certificates c
		JOIN courses co ON co.id = c.course_id
		JOIN users u ON u.id = c.user_id
		WHERE c.certificate_code = $1`, code).Scan(
		&cert.ID, &cert.UserID, &cert.CourseID, &cert.CertificateCode,
		&cert.ScorePct, &cert.Tier, &cert.IssuedAt, &cert.CourseTitle, &cert.UserName)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Certificate not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch certificate"})
		return
	}

	c.JSON(http.StatusOK, cert)
}

// GetCourseCertificate returns the certificate for the current user + the given course_id.
// GET /certificates/course/:course_id
func (h *Handler) GetCourseCertificate(c *gin.Context) {
	userID := c.GetInt64("user_id")
	courseIDStr := c.Param("course_id")
	courseID, err := strconv.ParseInt(courseIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID"})
		return
	}

	var cert CertificateWithCourse
	err = h.Pool.QueryRow(c.Request.Context(), `
		SELECT c.id, c.user_id, c.course_id, c.certificate_code, c.score_pct, c.tier, c.issued_at,
		       co.title AS course_title, u.name AS user_name
		FROM certificates c
		JOIN courses co ON co.id = c.course_id
		JOIN users u ON u.id = c.user_id
		WHERE c.user_id = $1 AND c.course_id = $2`, userID, courseID).Scan(
		&cert.ID, &cert.UserID, &cert.CourseID, &cert.CertificateCode,
		&cert.ScorePct, &cert.Tier, &cert.IssuedAt, &cert.CourseTitle, &cert.UserName)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Certificate not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch certificate"})
		return
	}

	c.JSON(http.StatusOK, cert)
}

// IssueForCourse issues a certificate for a completed course.
// POST /certificates/course/:course_id
func (h *Handler) IssueForCourse(c *gin.Context) {
	userID := c.GetInt64("user_id")
	courseIDStr := c.Param("course_id")
	courseID, err := strconv.ParseInt(courseIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID"})
		return
	}

	// Check that the user has completed the course
	enrollment, err := h.Queries.GetCourseEnrollment(c.Request.Context(),
		database.GetCourseEnrollmentParams{UserID: userID, CourseID: courseID})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Enrollment not found"})
		return
	}
	if enrollment.Status != "completed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Course not completed"})
		return
	}

	// Get the user's score for this course
	totalScore, err := h.Queries.GetUserTotalScore(c.Request.Context(), userID)
	if err != nil {
		totalScore = 0
	}
	if totalScore == nil {
		var zero interface{} = int64(0)
		totalScore = zero
	}
	// Default to 0, NOT 100. If there is no course_scores row we have no
	// evidence the learner earned anything; defaulting to 100 would hand out
	// unearned gold certificates. Legitimate completions always write a
	// course_scores row via ScoreCourse before this runs.
	scorePct := 0.0
	// Try to get the actual course score
	var courseScore int32
	err = h.Pool.QueryRow(c.Request.Context(),
		`SELECT score FROM course_scores WHERE user_id = $1 AND course_id = $2`,
		userID, courseID).Scan(&courseScore)
	if err != nil && err != pgx.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch course score"})
		return
	}
	if err == nil {
		scorePct = float64(courseScore)
	}

	cert, created, err := h.IssueCertificate(c.Request.Context(), userID, courseID, scorePct)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to issue certificate"})
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}

	c.JSON(status, cert)
}

// IssueCertificate issues a certificate when a user completes a course.
// Returns the certificate and whether a new one was created (false if already existed).
func (h *Handler) IssueCertificate(ctx context.Context, userID, courseID int64, scorePct float64) (*Certificate, bool, error) {
	// Compute tier
	tier := "bronze"
	if scorePct >= 90 {
		tier = "gold"
	} else if scorePct >= 75 {
		tier = "silver"
	}

	code := uuid.New().String()

	var cert Certificate
	var created bool
	err := h.Pool.QueryRow(ctx, `
		INSERT INTO certificates (user_id, course_id, certificate_code, score_pct, tier)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, course_id) DO UPDATE
			SET score_pct = EXCLUDED.score_pct,
			    tier = EXCLUDED.tier,
			    issued_at = NOW()
		RETURNING id, user_id, course_id, certificate_code, score_pct, tier, issued_at,
		          (xmax = 0) AS was_inserted`,
		userID, courseID, code, scorePct, tier).Scan(
		&cert.ID, &cert.UserID, &cert.CourseID, &cert.CertificateCode,
		&cert.ScorePct, &cert.Tier, &cert.IssuedAt, &created)
	if err != nil {
		return nil, false, fmt.Errorf("failed to upsert certificate: %w", err)
	}

	if created {
		audit.Logf(h.Queries, nil, "certificate_issued",
			"user %d earned tier=%s for course %d (score=%.1f%%)", userID, tier, courseID, scorePct)
	}

	return &cert, created, nil
}

// RegisterRoutesPublic registers public (no-auth) certificate routes.
func (h *Handler) RegisterRoutesPublic(r *gin.Engine) {
	r.GET("/api/certificates/:code", h.GetCertificateByCode)
}

// GetCourseCert returns the certificate for a user/course pair (programmatic, not an HTTP handler).
func (h *Handler) GetCourseCert(ctx context.Context, userID, courseID int64) *Certificate {
	var cert Certificate
	err := h.Pool.QueryRow(ctx, `
		SELECT id, user_id, course_id, certificate_code, score_pct, tier, issued_at
		FROM certificates
		WHERE user_id = $1 AND course_id = $2`, userID, courseID).Scan(
		&cert.ID, &cert.UserID, &cert.CourseID, &cert.CertificateCode,
		&cert.ScorePct, &cert.Tier, &cert.IssuedAt)
	if err != nil {
		return nil
	}
	return &cert
}

// RegisterRoutes registers protected certificate routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/certificates", h.ListCertificates)
	r.POST("/certificates/course/:course_id", h.IssueForCourse)
	r.GET("/certificates/course/:course_id", h.GetCourseCertificate)
}
