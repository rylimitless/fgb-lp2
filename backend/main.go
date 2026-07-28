package main

import (
	"context"
	"fgb-lp/adaptive"
	"fgb-lp/ai"
	"fgb-lp/analytics"
	"fgb-lp/app"
	"fgb-lp/badges"
	"fgb-lp/certificates"
	"fgb-lp/coach"
	content_repository "fgb-lp/content_repository"
	database "fgb-lp/database/queries"
	"fgb-lp/departments"
	"fgb-lp/documents"
	"fgb-lp/enrollments"
	"fgb-lp/gamification"
	homehandler "fgb-lp/home_handler"
	"fgb-lp/learning_paths"
	"fgb-lp/lessons"
	"fgb-lp/mailer"
	"fgb-lp/middlewares"
	"fgb-lp/notifications"
	"fgb-lp/password_reset"
	"fgb-lp/review"
	"fgb-lp/users"
	"fgb-lp/worker"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {

	godotenv.Load()

	dbpool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}

	defer dbpool.Close()

	queries := database.New(dbpool)

	// Run startup migrations for schema changes that may not be applied
	// (docker-entrypoint-initdb.d only runs on first container creation)
	runMigrations(dbpool)

	app := &app.App{
		Pool:    dbpool,
		Queries: queries,
	}

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3039", "https://fgbacademy.rybuildstuff.dev", "https://fgbguide.rybuildstuff.dev", "https://98fe-173-225-243-241.ngrok-free.app"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	r.Use(gin.Logger())

	r.GET("/api/check-first-user", func(c *gin.Context) {
		isFirst := app.CheckIfFirstUser()
		c.JSON(http.StatusOK, gin.H{"first_user": isFirst})
	})

	r.GET("/api/health", func(c *gin.Context) {

		c.JSON(http.StatusOK, gin.H{"Status": "Ok"})
	})

	r.POST("/api/setup", func(c *gin.Context) {
		var body struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required,min=8"`
			Name     string `json:"name" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		token, err := app.SetupAdmin(c.Request.Context(), body.Email, body.Password, body.Name)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		setSessionCookie(c, token, 86400)
		c.JSON(http.StatusCreated, gin.H{"message": "admin created"})
	})

	r.POST("/api/login", func(c *gin.Context) {
		var body struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		token, err := app.Login(c.Request.Context(), body.Email, body.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
			return
		}

		setSessionCookie(c, token, 86400)
		c.JSON(http.StatusOK, gin.H{"message": "logged in"})
	})

	protected := r.Group("/api")

	protected.Use(middlewares.RequireAuth(app.Queries))

	// Role-gated sub-groups for API route protection
	adminGroup := protected.Group("")
	adminGroup.Use(middlewares.RequireRole("admin"))

	adminManagerGroup := protected.Group("")
	adminManagerGroup.Use(middlewares.RequireRole("admin", "manager"))

	approverGroup := protected.Group("")
	approverGroup.Use(middlewares.RequireRole("approver"))

	r.POST("/api/logout", func(c *gin.Context) {
		token, _ := c.Cookie("session_token")
		_ = app.Logout(c.Request.Context(), token)
		setSessionCookie(c, "", -1)
		c.JSON(http.StatusOK, gin.H{"message": "logged out"})
	})

	// Returns the list of available roles and their descriptions for frontend config
	protected.GET("/roles", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"roles": []gin.H{
				{"id": "admin", "label": "Admin", "description": "Full system access"},
				{"id": "end user", "label": "End User", "description": "Access to Gia Coach, courses, and adaptive learning"},
				{"id": "content creator", "label": "Content Creator", "description": "Create and upload content and courses, plus all end-user features"},
				{"id": "approver", "label": "Approver", "description": "Approve, request changes, or reject content and courses, plus end-user features"},
				{"id": "manager", "label": "Manager", "description": "View analytics and manage users"},
				{"id": "auditor", "label": "Auditor", "description": "View-only access to reports and users"},
			},
		})
	})

	protected.GET("/me", func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		user, err := queries.GetUserByID(c.Request.Context(), userID.(int64))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		// Get all roles from user_roles (with fallback to single role)
		roles, _ := middlewares.GetUserRoles(c, queries, userID.(int64))

		c.JSON(http.StatusOK, gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.Role,
			"roles": roles,
		})
	})

	uploadDir := "uploads"
	if err := worker.CreateUploadDir(uploadDir); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create upload directory: %v\n", err)
		os.Exit(1)
	}

	// Create the shared email sender
	emailSender := mailer.NewResend()

	docHandler := documents.NewHandler(queries, uploadDir)
	docHandler.RegisterRoutes(protected)

	aiHandler := ai.NewHandler(dbpool, queries)
	aiHandler.RegisterRoutes(protected)

	reviewHandler := review.NewHandler(queries).WithMailer(emailSender)
	reviewHandler.RegisterRoutes(approverGroup)

	// Certificates and badges — created first so they can be wired into other handlers
	certHandler := certificates.NewHandler(queries, dbpool)
	certHandler.RegisterRoutes(protected)
	certHandler.RegisterRoutesPublic(r)

	badgesHandler := badges.NewHandler(queries, dbpool)
	badgesHandler.RegisterRoutes(protected)

	// Lessons handler with certificates + badges + mailer for auto-issuing on course completion
	lessonHandler := lessons.NewHandler(queries).
		WithCertificates(certHandler).
		WithBadges(badgesHandler).
		WithMailer(emailSender)
	lessonHandler.RegisterRoutes(protected)

	gamificationHandler := gamification.NewHandler(queries)
	gamificationHandler.RegisterRoutes(protected)

	coachHandler := coach.NewHandler(queries)
	coachHandler.RegisterRoutes(protected)

	adaptiveHandler := adaptive.NewHandler(queries)
	adaptiveHandler.RegisterRoutes(protected)

	repoHandler := content_repository.NewHandler(dbpool, queries)
	repoHandler.RegisterRoutes(protected)

	notifHandler := notifications.NewHandler(queries, dbpool)
	notifHandler.RegisterRoutes(protected)

	enrollmentHandler := enrollments.NewHandler(queries).WithMailer(emailSender)
	enrollmentHandler.RegisterRoutes(protected)
	enrollmentHandler.RegisterAdminRoutes(adminManagerGroup)

	dashHandler := homehandler.NewDashboardHandler(dbpool)
	protected.GET("/dashboard", dashHandler.GetDashboard)

	userHandler := users.NewHandler(dbpool, queries).WithMailer(emailSender)
	userHandler.RegisterRoutes(adminGroup)
	userHandler.RegisterListRoute(adminManagerGroup)

	lpHandler := learning_paths.NewHandler(queries)
	lpHandler.RegisterRoutes(adminManagerGroup)
	lpHandler.RegisterLearnerRoutes(protected)

	deptHandler := departments.NewHandler(queries)
	deptHandler.RegisterRoutes(adminManagerGroup)

	analyticsHandler := analytics.NewHandler(queries)
	analyticsHandler.RegisterRoutes(adminGroup)

	// Password reset (public routes — no auth required)
	resetHandler := password_reset.NewHandler(dbpool, queries).WithMailer(emailSender)
	resetHandler.RegisterRoutes(r.Group("/api"))

	wrk := worker.New(queries, uploadDir)
	go wrk.Start(context.Background())

	r.Run(":5555")
}

func setSessionCookie(c *gin.Context, token string, maxAge int) {
	c.SetCookie("session_token", token, maxAge, "/", "", false, true)
}

func runMigrations(dbpool *pgxpool.Pool) {
	ctx := context.Background()
	migrations := []string{
		// Certificates and badges migrations are in scripts/01_certificates_badges_v1.sql
		// (run manually or via docker-entrypoint-initdb.d)

		// Staged course builder: track per-module generation status separately
		// from the course's overall draft/published status. Allows modules to
		// be generated one at a time after the outline is approved.
		`alter table modules add column if not exists status text not null default 'pending'
			check (status in ('pending', 'generating', 'ready', 'failed'))`,

		// Per-module question type allowlist + AI-reported limitations.
		`alter table modules add column if not exists question_types jsonb`,
		`alter table modules add column if not exists limitations jsonb`,

		// Staged course builder: distinguish one-shot 'full' jobs from 'outline'
		// and per-module jobs, and remember which module a module-stage job owns.
		`alter table course_generation_jobs add column if not exists stage text not null default 'full'
			check (stage in ('full', 'outline', 'module'))`,
		`alter table course_generation_jobs add column if not exists module_id bigint references modules(id) on delete set null`,
	}

	for _, m := range migrations {
		if _, err := dbpool.Exec(ctx, m); err != nil {
			fmt.Fprintf(os.Stderr, "Migration warning: %v\n", err)
		}
	}
}
