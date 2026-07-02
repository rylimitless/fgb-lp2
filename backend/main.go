package main

import (
	"context"
	"fgb-lp/adaptive"
	"fgb-lp/ai"
	"fgb-lp/analytics"
	"fgb-lp/app"
	"fgb-lp/coach"
	"fgb-lp/content_repository"
	database "fgb-lp/database/queries"
	"fgb-lp/departments"
	"fgb-lp/documents"
	"fgb-lp/enrollments"
	"fgb-lp/gamification"
	homehandler "fgb-lp/home_handler"
	"fgb-lp/learning_paths"
	"fgb-lp/lessons"
	"fgb-lp/middlewares"
	"fgb-lp/notifications"
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

	docHandler := documents.NewHandler(queries, uploadDir)
	docHandler.RegisterRoutes(protected)

	aiHandler := ai.NewHandler(dbpool, queries)
	aiHandler.RegisterRoutes(protected)

	reviewHandler := review.NewHandler(queries)
	reviewHandler.RegisterRoutes(approverGroup)

	lessonHandler := lessons.NewHandler(queries)
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

	enrollmentHandler := enrollments.NewHandler(queries)
	enrollmentHandler.RegisterRoutes(protected)
	enrollmentHandler.RegisterAdminRoutes(adminManagerGroup)

	dashHandler := homehandler.NewDashboardHandler(dbpool)
	protected.GET("/dashboard", dashHandler.GetDashboard)

	userHandler := users.NewHandler(dbpool, queries)
	userHandler.RegisterRoutes(adminGroup)
	userHandler.RegisterListRoute(adminManagerGroup)

	lpHandler := learning_paths.NewHandler(queries)
	lpHandler.RegisterRoutes(adminManagerGroup)
	lpHandler.RegisterLearnerRoutes(protected)

	deptHandler := departments.NewHandler(queries)
	deptHandler.RegisterRoutes(adminManagerGroup)

	analyticsHandler := analytics.NewHandler(queries)
	analyticsHandler.RegisterRoutes(adminGroup)

	wrk := worker.New(queries, uploadDir)
	go wrk.Start(context.Background())

	r.Run(":5555")
}

// setSessionCookie writes (or clears) the session cookie with attributes that
// survive both the Vite dev proxy and the production Caddy reverse proxy.
//
// Cookie attributes here are the root cause of "logged out after a few seconds"
// symptoms: the backend previously set Secure=false with no explicit SameSite.
// Browsers default that to SameSite=Lax and, in strict/HTTPS contexts, evict or
// refuse the cookie once the response is relayed through a reverse proxy.
//
// We set SameSite=Lax explicitly and derive Secure from the effective request
// scheme (X-Forwarded-Proto is set by Caddy in prod; the Vite dev proxy runs
// over HTTP so Secure stays false, which is correct for localhost).
func setSessionCookie(c *gin.Context, value string, maxAge int) {
	secure := c.GetHeader("X-Forwarded-Proto") == "https" || c.Request.TLS != nil
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("session_token", value, maxAge, "/", "", secure, true)
}

// runMigrations applies schema changes that haven't been applied via docker-entrypoint.
func runMigrations(pool *pgxpool.Pool) {
	ctx := context.Background()

	// Migrate document_chunks.embedding from vector(4096) (OpenRouter qwen3-embedding-8b)
	// to vector(384) (local all-MiniLM-L6-v2 via fastembed). Dimensions are incompatible,
	// so existing embeddings are cleared (NULL) — re-ingest documents to repopulate them.
	_, err := pool.Exec(ctx, `
DO $$
DECLARE
  col_type text;
BEGIN
  SELECT format_type(a.atttypid, a.atttypmod)
  INTO col_type
  FROM pg_attribute a
  WHERE a.attrelid = 'document_chunks'::regclass
    AND a.attname = 'embedding';

  IF col_type = 'vector(4096)' THEN
    UPDATE document_chunks SET embedding = NULL;
    ALTER TABLE document_chunks ALTER COLUMN embedding TYPE vector(384);
  END IF;
END $$;
	`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migration document_chunks embedding dimension: %v\n", err)
	}

	// Fix course_items constraint to allow 'content' type
	_, err = pool.Exec(ctx, `
DO $$
DECLARE
  constraint_name text;
BEGIN
  SELECT con.conname INTO constraint_name
  FROM pg_constraint con
  JOIN pg_class rel ON rel.oid = con.conrelid
  WHERE rel.relname = 'course_items'
    AND con.contype = 'c'
    AND pg_get_constraintdef(con.oid) LIKE '%item_type%';
  IF constraint_name IS NOT NULL THEN
    EXECUTE 'ALTER TABLE course_items DROP CONSTRAINT ' || constraint_name;
  END IF;
END $$;

ALTER TABLE course_items DROP CONSTRAINT IF EXISTS course_items_item_type_check;
ALTER TABLE course_items ADD CONSTRAINT course_items_item_type_check
  CHECK (item_type IN ('content','mc','ma','tf','fb','sa','matching','drag_sort','hotspot','sequence','scale'));
	`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migration course_items constraint: %v\n", err)
	}

	// Create jobs table if it doesn't exist
	_, err = pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS course_generation_jobs (
  id text primary key,
  status text not null default 'pending'
    check (status in ('pending', 'running', 'completed', 'failed')),
  request jsonb not null,
  steps jsonb not null default '[]',
  modules jsonb not null default '[]',
  result jsonb,
  error text,
  course_id bigint references courses(id) on delete set null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
	`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migration jobs table: %v\n", err)
	}

	// Create notifications table if it doesn't exist
	_, err = pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS notifications (
  id bigserial primary key,
  user_id bigint references users(id) on delete cascade,
  title text not null,
  message text not null default '',
  link text not null default '',
  is_read boolean not null default false,
  created_at timestamptz not null default now()
);
CREATE INDEX IF NOT EXISTS idx_notifications_user_unread
  on notifications(user_id, is_read) where is_read = false;
	`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migration notifications table: %v\n", err)
	}

	// Create coach_queries table if it doesn't exist
	_, err = pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS coach_queries (
  id bigserial primary key,
  user_id bigint references users(id) on delete set null,
  question text not null,
  sources_count int not null default 0,
  created_at timestamptz not null default now()
);
CREATE INDEX IF NOT EXISTS idx_coach_queries_created_at on coach_queries(created_at);
	`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migration coach_queries table: %v\n", err)
	}

	// Create user_roles junction table for many-to-many role assignments
	_, err = pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS user_roles (
  user_id bigint not null references users(id) on delete cascade,
  role text not null,
  primary key (user_id, role)
);
	`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migration user_roles table: %v\n", err)
	}

	// Seed role_permissions for all roles
	_, err = pool.Exec(ctx, `
INSERT INTO permissions (name) VALUES
  ('content:upload'),
  ('content:create_course'),
  ('content:edit_course'),
  ('content:delete'),
  ('content:approve'),
  ('content:reject'),
  ('content:request_changes'),
  ('users:manage'),
  ('users:view'),
  ('gia:talk'),
  ('courses:take'),
  ('adaptive:practice'),
  ('analytics:view')
ON CONFLICT (name) DO NOTHING;

-- Admin: all permissions
INSERT INTO role_permissions (role, permission_id)
SELECT 'admin', id FROM permissions
ON CONFLICT (role, permission_id) DO NOTHING;

-- End user: talk to Gia, take courses, adaptive learning
INSERT INTO role_permissions (role, permission_id)
SELECT 'end user', id FROM permissions WHERE name IN (
  'gia:talk', 'courses:take', 'adaptive:practice'
)
ON CONFLICT (role, permission_id) DO NOTHING;

-- Content creator: end user + create/upload/edit/delete content
INSERT INTO role_permissions (role, permission_id)
SELECT 'content creator', id FROM permissions WHERE name IN (
  'gia:talk', 'courses:take', 'adaptive:practice',
  'content:upload', 'content:create_course', 'content:edit_course', 'content:delete'
)
ON CONFLICT (role, permission_id) DO NOTHING;

-- Approver: end user + approve/reject/request changes but NOT create/upload
INSERT INTO role_permissions (role, permission_id)
SELECT 'approver', id FROM permissions WHERE name IN (
  'gia:talk', 'courses:take', 'adaptive:practice',
  'content:approve', 'content:reject', 'content:request_changes'
)
ON CONFLICT (role, permission_id) DO NOTHING;

-- Manager: view everything, manage users
INSERT INTO role_permissions (role, permission_id)
SELECT 'manager', id FROM permissions WHERE name IN (
  'gia:talk', 'courses:take', 'adaptive:practice',
  'users:manage', 'users:view', 'analytics:view'
)
ON CONFLICT (role, permission_id) DO NOTHING;

-- Auditor: view-only across the board
INSERT INTO role_permissions (role, permission_id)
SELECT 'auditor', id FROM permissions WHERE name IN (
  'users:view', 'analytics:view'
)
ON CONFLICT (role, permission_id) DO NOTHING;

-- Migrate existing single-role users into user_roles table
INSERT INTO user_roles (user_id, role)
SELECT id, role FROM users
ON CONFLICT (user_id, role) DO NOTHING;
		`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migration role_permissions seed: %v\n", err)
	}

	// Create audit_log table if it doesn't exist
	_, err = pool.Exec(ctx, `
	CREATE TABLE IF NOT EXISTS audit_log (
	  id bigserial primary key,
	  user_id bigint references users(id) on delete set null,
	  action text not null,
	  details jsonb not null default '{}',
	  created_at timestamptz not null default now()
	);
	CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON audit_log(created_at desc);
	CREATE INDEX IF NOT EXISTS idx_audit_log_user_id ON audit_log(user_id);
	CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log(action);
		`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migration audit_log table: %v\n", err)
	}

	// Create user_streaks table for daily engagement tracking
	_, err = pool.Exec(ctx, `
	CREATE TABLE IF NOT EXISTS user_streaks (
	  user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	  streak_date date NOT NULL,
	  created_at timestamptz NOT NULL DEFAULT now(),
	  PRIMARY KEY (user_id, streak_date)
	);
	CREATE INDEX IF NOT EXISTS idx_user_streaks_user_date ON user_streaks(user_id, streak_date desc);
		`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migration user_streaks table: %v\n", err)
	}

	// Create course_scores table for leaderboard scoring
	_, err = pool.Exec(ctx, `
	CREATE TABLE IF NOT EXISTS course_scores (
	  user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	  course_id bigint NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
	  score int NOT NULL DEFAULT 0,
	  completed_at timestamptz,
	  PRIMARY KEY (user_id, course_id)
	);
	CREATE INDEX IF NOT EXISTS idx_course_scores_score ON course_scores(score desc);
	CREATE INDEX IF NOT EXISTS idx_course_scores_course ON course_scores(course_id);
		`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migration course_scores table: %v\n", err)
	}

	// Create enrollments table for user-course lifecycle management
	_, err = pool.Exec(ctx, `
	ALTER TABLE courses ADD COLUMN IF NOT EXISTS capacity int;

	CREATE TABLE IF NOT EXISTS enrollments (
	  id bigserial primary key,
	  user_id bigint not null references users(id) on delete cascade,
	  course_id bigint not null references courses(id) on delete cascade,
	  status text not null default 'active'
	    check (status in ('active', 'completed', 'dropped', 'pending')),
	  progress_pct numeric(5,2) not null default 0,
	  enrolled_at timestamptz not null default now(),
	  completed_at timestamptz,
	  dropped_at timestamptz,
	  unique(user_id, course_id)
	);
	CREATE INDEX IF NOT EXISTS idx_enrollments_user ON enrollments(user_id);
	CREATE INDEX IF NOT EXISTS idx_enrollments_course ON enrollments(course_id);
	CREATE INDEX IF NOT EXISTS idx_enrollments_status ON enrollments(status);

	-- Backfill: create enrollment records for users who already have lesson_progress
	INSERT INTO enrollments (user_id, course_id, status, progress_pct, enrolled_at, completed_at)
	SELECT lp.user_id, lp.course_id,
	  CASE WHEN lp.completed THEN 'completed'::text ELSE 'active'::text END,
	  coalesce(lp.score_pct, 0),
	  lp.started_at,
	  lp.completed_at
	FROM lesson_progress lp
	ON CONFLICT (user_id, course_id) DO NOTHING;
		`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migration enrollments table: %v\n", err)
	}
}
