package password_reset

import (
	"context"
	"fgb-lp/audit"
	database "fgb-lp/database/queries"
	"fgb-lp/functions"
	"fgb-lp/mailer"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	Pool    *pgxpool.Pool
	Queries *database.Queries
	Mailer  mailer.Sender
}

func NewHandler(pool *pgxpool.Pool, queries *database.Queries) *Handler {
	return &Handler{Pool: pool, Queries: queries}
}

func (h *Handler) WithMailer(m mailer.Sender) *Handler {
	h.Mailer = m
	return h
}

// ForgotPassword accepts an email and sends a password reset link if the user exists.
// Always returns 200 OK to prevent email enumeration.
func (h *Handler) ForgotPassword(c *gin.Context) {
	var body struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide a valid email address."})
		return
	}

	user, err := h.Queries.GetUserByEmail(c.Request.Context(), body.Email)
	if err != nil {
		// Don't reveal whether the email exists — but do log it
		log.Printf("[password_reset] reset requested for unknown email: %s", body.Email)
		c.JSON(http.StatusOK, gin.H{"message": "If that email is registered, a reset link has been sent."})
		return
	}

	token, err := functions.MakeTokens()
	if err != nil {
		log.Printf("[password_reset] failed to generate token: %v", err)
		c.JSON(http.StatusOK, gin.H{"message": "If that email is registered, a reset link has been sent."})
		return
	}
	_, err = h.Queries.CreatePasswordResetToken(c.Request.Context(), database.CreatePasswordResetTokenParams{
		UserID: user.ID,
		Token:  token,
	})
	if err != nil {
		log.Printf("[password_reset] failed to create token for user %d: %v", user.ID, err)
		c.JSON(http.StatusOK, gin.H{"message": "If that email is registered, a reset link has been sent."})
		return
	}

	// Build the reset URL from a server-controlled APP_URL. Never trust the
	// client Origin header — an attacker could poison the link and steal the
	// token when the victim clicks it.
	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "https://fgbacademy.rybuildstuff.dev"
	}
	resetURL := appURL + "/reset-password?token=" + token

	if h.Mailer != nil {
		go func() {
			if err := h.Mailer.SendPasswordReset(context.Background(), user.Email, user.Name, resetURL); err != nil {
				log.Printf("[password_reset] failed to send reset email to %s: %v", user.Email, err)
				audit.Logf(h.Queries, nil, "email_password_reset_failed", "to=%s error=%v", user.Email, err)
			} else {
				audit.Logf(h.Queries, nil, "email_password_reset_sent", "to=%s", user.Email)
			}
		}()
	}

	audit.Log(h.Queries, c, "password_reset_requested", map[string]any{
		"user_id": user.ID,
		"email":   user.Email,
	})

	c.JSON(http.StatusOK, gin.H{"message": "If that email is registered, a reset link has been sent."})
}

// ResetPassword accepts a token and new password, then updates the user's password.
func (h *Handler) ResetPassword(c *gin.Context) {
	var body struct {
		Token    string `json:"token" binding:"required"`
		Password string `json:"password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 8 characters."})
		return
	}

	resetToken, err := h.Queries.GetPasswordResetToken(c.Request.Context(), body.Token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired reset link. Please request a new one."})
		return
	}

	hash := functions.MakeHash(body.Password)
	_, err = h.Queries.UpdateUserPassword(c.Request.Context(), database.UpdateUserPasswordParams{
		ID:           resetToken.UserID,
		PasswordHash: hash,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password."})
		return
	}

	// Mark token as used so it can't be reused
	_ = h.Queries.MarkPasswordResetTokenUsed(c.Request.Context(), body.Token)

	// Invalidate all existing sessions for this user (force re-login)
	_ = h.Queries.DeleteUserSessions(c.Request.Context(), resetToken.UserID)

	audit.Log(h.Queries, c, "password_reset_completed", map[string]any{
		"user_id": resetToken.UserID,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Password has been reset. You can now sign in with your new password."})
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/forgot-password", h.ForgotPassword)
	r.POST("/reset-password", h.ResetPassword)
}
