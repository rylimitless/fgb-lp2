package main

import (
	"context"
	"fgb-lp/app"
	database "fgb-lp/database/queries"
	"fgb-lp/documents"
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

	app := &app.App{
		Pool:    dbpool,
		Queries: queries,
	}

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	r.Use(gin.Logger())

	// ----- public routes (no auth required) -----
	r.GET("/api/check-first-user", func(c *gin.Context) {
		isFirst := app.CheckIfFirstUser()
		c.JSON(http.StatusOK, gin.H{"first_user": isFirst})
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

		c.SetCookie("session_token", token, 86400, "/", "", false, true)
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

		c.SetCookie("session_token", token, 86400, "/", "", false, true)
		c.JSON(http.StatusOK, gin.H{"message": "logged in"})
	})

	// ----- protected routes -----
	protected := r.Group("/api")
	// protected.Use(func(c *gin.Context) {
	// 	token, err := c.Cookie("session_token")
	// 	if err != nil {
	// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
	// 		c.Abort()
	// 		return
	// 	}

	// 	session, err := queries.GetSessionByToken(c.Request.Context(), token)
	// 	if err != nil {
	// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
	// 		c.Abort()
	// 		return
	// 	}

	// 	c.Set("user_id", session.UserID)
	// 	c.Next()
	// })

	protected.POST("/logout", func(c *gin.Context) {
		token, _ := c.Cookie("session_token")
		_ = app.Logout(c.Request.Context(), token)
		c.SetCookie("session_token", "", -1, "/", "", false, true)
		c.JSON(http.StatusOK, gin.H{"message": "logged out"})
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
		c.JSON(http.StatusOK, gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.Role,
		})
	})

	// ----- document routes -----
	uploadDir := "uploads"
	if err := worker.CreateUploadDir(uploadDir); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create upload directory: %v\n", err)
		os.Exit(1)
	}

	docHandler := documents.NewHandler(queries, uploadDir)
	docHandler.RegisterRoutes(protected)

	// ----- start background worker -----
	wrk := worker.New(queries, uploadDir)
	go wrk.Start(context.Background())

	r.Run(":5555")
}
