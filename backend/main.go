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
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3039", "https://fgbacademy.rybuildstuff.dev", "https://fgbguide.rybuildstuff.dev"},
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

	r.POST("/api/setup", middlewares.RateLimitByIP(5, 1), func(c *gin.Context) {
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

	r.POST("/api/login", middlewares.RateLimitByIP(10, 5), func(c *gin.Context) {
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

	auditorGroup := protected.Group("")
	auditorGroup.Use(middlewares.RequireRole("auditor"))

	// Routes accessible by admin, auditor, or manager (read-only operational views)
	auditorManagerGroup := protected.Group("")
	auditorManagerGroup.Use(middlewares.RequireRole("admin", "auditor", "manager"))

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
	// Developer tools: mock-course generator + LLM spend dashboard. Admin only —
	// these routes create real course rows and incur real model cost.
	aiHandler.RegisterDevRoutes(adminGroup)

	reviewHandler := review.NewHandler(queries).WithMailer(emailSender)
	reviewHandler.RegisterRoutes(approverGroup)

	// Certificates and badges — created first so they can be wired into other handlers
	certHandler := certificates.NewHandler(queries, dbpool)
	certHandler.RegisterRoutes(protected)
	certHandler.RegisterRoutesPublic(r)

	badgesHandler := badges.NewHandler(queries, dbpool)
	badgesHandler.RegisterRoutes(protected)

	// Learning paths handler — created before lessons so it can be wired in as a
	// path-progress notifier (course completions cascade into path progress).
	lpHandler := learning_paths.NewHandler(queries, dbpool)
	lpHandler.RegisterRoutes(adminManagerGroup)
	lpHandler.RegisterLearnerRoutes(protected)

	// Lessons handler with certificates + badges + mailer + path-progress for auto-issuing on course completion
	lessonHandler := lessons.NewHandler(queries).
		WithPool(dbpool).
		WithCertificates(certHandler).
		WithBadges(badgesHandler).
		WithMailer(emailSender).
		WithPathProgress(lpHandler)
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

	enrollmentHandler := enrollments.NewHandler(queries).WithPool(dbpool).WithMailer(emailSender)
	enrollmentHandler.RegisterRoutes(protected)
	enrollmentHandler.RegisterAdminRoutes(adminManagerGroup)

	dashHandler := homehandler.NewDashboardHandler(dbpool)
	protected.GET("/dashboard", dashHandler.GetDashboard)

	userHandler := users.NewHandler(dbpool, queries).WithMailer(emailSender)
	userHandler.RegisterRoutes(adminGroup)
	userHandler.RegisterListRoute(auditorManagerGroup) // admin, auditor, manager all get read-only user list

	deptHandler := departments.NewHandler(queries)
	deptHandler.RegisterRoutes(adminManagerGroup)

	analyticsHandler := analytics.NewHandler(queries, dbpool)
	analyticsHandler.RegisterRoutes(auditorManagerGroup)        // admin, auditor, manager all get analytics
	analyticsHandler.RegisterManagerRoutes(auditorManagerGroup) // team-focused analytics

	// Password reset (public routes — no auth required). Rate-limited per IP to
	// prevent email-bombing via forgot-password (fixes audit item H3).
	resetHandler := password_reset.NewHandler(dbpool, queries).WithMailer(emailSender)
	resetRoutes := r.Group("/api")
	resetRoutes.Use(middlewares.RateLimitByIP(3, 1))
	resetHandler.RegisterRoutes(resetRoutes)

	wrk := worker.New(queries, uploadDir)
	go wrk.Start(context.Background())

	r.Run(":5555")
}

func setSessionCookie(c *gin.Context, token string, maxAge int) {
	secure := os.Getenv("SECURE_COOKIE") == "true"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("session_token", token, maxAge, "/", "", secure, true)
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

		// Adaptive learning by question type: items that test the SAME concept
		// but in different formats (mc/tf/fb/...) share a question_group_id so
		// the adaptive engine can serve each learner their preferred/best type
		// for a given concept instead of always the same format.
		`alter table course_items add column if not exists question_group_id text`,
		`create index if not exists course_items_group_idx on course_items (course_id, question_group_id)`,

		// Per-type performance tracking so "responds better to a certain type"
		// is data-driven, not just a static preference. type_stats is a JSONB
		// map of item_type -> {answered, correct}. preferred_type is an explicit
		// override the learner (or admin) can set.
		`alter table learning_preferences add column if not exists preferred_type text`,
		`alter table learning_preferences add column if not exists type_stats jsonb not null default '{}'`,

		// pgvector index for semantic retrieval (coach + AI course generation).
		// HNSW is preferred over IVFFlat: no training step, good recall at small
		// scale, and works fine as rows are added. Without this every embedding
		// query does a brute-force linear scan.
		`create index if not exists document_chunks_embedding_hnsw
			on document_chunks using hnsw (embedding vector_cosine_ops)`,

		// Allow documents to be marked 'rejected' during review (previously the
		// approve/reject handler always set status='ready').
		`alter table documents drop constraint if exists documents_status_check`,
		`alter table documents add constraint documents_status_check
			check (status in ('uploaded', 'processing', 'ready', 'failed', 'rejected'))`,

		// Developer tooling: per-call LLM token accounting. Every chat completion
		// logs one row here so the "mock course" spend dashboard can compute
		// averages across runs. Safe to run repeatedly (if not exists).
		`create table if not exists llm_token_usage (
		   id bigserial primary key,
		   model text not null,
		   source text not null default 'unknown',
		   label text not null default '',
		   prompt_tokens bigint not null default 0,
		   completion_tokens bigint not null default 0,
		   total_tokens bigint not null default 0,
		   cached_prompt_tokens bigint not null default 0,
		   reasoning_tokens bigint not null default 0,
		   course_id bigint references courses(id) on delete set null,
		   user_id bigint references users(id) on delete set null,
		   job_ref text,
		   created_at timestamptz not null default now()
		)`,
		`create index if not exists idx_llm_token_usage_created_at on llm_token_usage(created_at desc)`,
		`create index if not exists idx_llm_token_usage_source on llm_token_usage(source)`,
		`create index if not exists idx_llm_token_usage_job_ref on llm_token_usage(job_ref)`,
		`create index if not exists idx_llm_token_usage_course_id on llm_token_usage(course_id)`,
	}

	for _, m := range migrations {
		if _, err := dbpool.Exec(ctx, m); err != nil {
			fmt.Fprintf(os.Stderr, "Migration warning: %v\n", err)
		}
	}
}
