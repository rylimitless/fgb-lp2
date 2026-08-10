package notifications

import (
	"context"
	"encoding/json"
	database "fgb-lp/database/queries"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
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

// ListNotifications returns recent notifications + unread count for the current user.
func (h *Handler) ListNotifications(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := pgtype.Int8{Int64: userID.(int64), Valid: true}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	notifs, err := h.Queries.GetUserNotifications(c.Request.Context(),
		database.GetUserNotificationsParams{
			UserID: uid,
			Limit:  int32(limit),
		})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}
	if notifs == nil {
		notifs = []database.Notification{}
	}

	unread, _ := h.Queries.CountUnreadNotifications(c.Request.Context(), uid)

	c.JSON(http.StatusOK, gin.H{
		"items":  notifs,
		"unread": unread,
	})
}

// MarkRead marks a single notification as read. Scoped to the current user
// so a learner can't suppress someone else's notification by guessing its id
// (fixes audit item H2).
func (h *Handler) MarkRead(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	userID, _ := c.Get("user_id")
	uid := userID.(int64)

	tag, err := h.Pool.Exec(c.Request.Context(),
		`UPDATE notifications SET is_read = true WHERE id = $1 AND user_id = $2`,
		id, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark as read"})
		return
	}
	if tag.RowsAffected() == 0 {
		// Either the notification doesn't exist or it belongs to another user.
		// Return 404 rather than leaking which.
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// UnreadCount returns just the unread count (for badge polling).
func (h *Handler) UnreadCount(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := pgtype.Int8{Int64: userID.(int64), Valid: true}
	count, _ := h.Queries.CountUnreadNotifications(c.Request.Context(), uid)
	c.JSON(http.StatusOK, gin.H{"unread": count})
}

// MarkAllRead flips every unread notification for the current user to read.
// Uses raw SQL through the pool because the auto-generated queries do not
// (yet) expose a bulk mark endpoint.
func (h *Handler) MarkAllRead(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(int64)
	tag, err := h.Pool.Exec(c.Request.Context(),
		`UPDATE notifications SET is_read = true WHERE user_id = $1 AND is_read = false`, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notifications as read"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated": tag.RowsAffected()})
}

// CreateForAdmins creates a notification for every admin user.
func (h *Handler) CreateForAdmins(title, message, link string) {
	admins, err := h.Queries.GetAdminUsers(context.Background())
	if err != nil {
		return
	}
	for _, admin := range admins {
		h.Queries.CreateNotification(context.Background(), database.CreateNotificationParams{
			UserID:  pgtype.Int8{Int64: admin.ID, Valid: true},
			Title:   title,
			Message: message,
			Link:    link,
		})
	}
}

// CreateForAll creates a notification for every user.
func (h *Handler) CreateForAll(title, message, link string) {
	users, err := h.Queries.GetAllUsers(context.Background())
	if err != nil {
		return
	}
	for _, user := range users {
		h.Queries.CreateNotification(context.Background(), database.CreateNotificationParams{
			UserID:  pgtype.Int8{Int64: user.ID, Valid: true},
			Title:   title,
			Message: message,
			Link:    link,
		})
	}
}

// CheckDeadlines scans all active enrollments with days_to_complete set and creates
// notifications for approaching or overdue deadlines. Safe to call on every dashboard load;
// uses the notification table's natural dedup (idempotent per unique title+user at worst).
func (h *Handler) CheckDeadlines(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(int64)
	ctx := c.Request.Context()

	rows, err := h.Pool.Query(ctx, `
		SELECT e.id, c.id, c.title, c.settings, e.enrolled_at
		FROM enrollments e
		JOIN courses c ON c.id = e.course_id
		WHERE e.user_id = $1 AND e.status = 'active'
	`, uid)
	if err != nil {
		return
	}
	defer rows.Close()

	now := time.Now()
	for rows.Next() {
		var enrollmentID, courseID int64
		var title string
		var settingsJSON []byte
		var enrolledAt time.Time
		if err := rows.Scan(&enrollmentID, &courseID, &title, &settingsJSON, &enrolledAt); err != nil {
			continue
		}

		var settings map[string]interface{}
		if err := json.Unmarshal(settingsJSON, &settings); err != nil {
			continue
		}
		dtc, ok := settings["days_to_complete"].(float64)
		if !ok || dtc <= 0 {
			continue
		}

		deadline := enrolledAt.Add(time.Duration(int(dtc)) * 24 * time.Hour)
		daysLeft := int(deadline.Sub(now).Hours() / 24)

		link := fmt.Sprintf("/lesson-player?id=%d", courseID)

		// Only notify at specific milestones: overdue, 1 day, 3 days
		if daysLeft < 0 {
			h.Queries.CreateNotification(ctx, database.CreateNotificationParams{
				UserID:  pgtype.Int8{Int64: uid, Valid: true},
				Title:   fmt.Sprintf("Deadline passed: %s", title),
				Message: fmt.Sprintf("Your enrollment in \"%s\" is overdue by %d day(s). Please complete it as soon as possible.", title, -daysLeft),
				Link:    link,
			})
		} else if daysLeft <= 1 {
			h.Queries.CreateNotification(ctx, database.CreateNotificationParams{
				UserID:  pgtype.Int8{Int64: uid, Valid: true},
				Title:   fmt.Sprintf("Deadline tomorrow: %s", title),
				Message: fmt.Sprintf("You have %d day(s) left to complete \"%s\".", daysLeft, title),
				Link:    link,
			})
		} else if daysLeft <= 3 {
			h.Queries.CreateNotification(ctx, database.CreateNotificationParams{
				UserID:  pgtype.Int8{Int64: uid, Valid: true},
				Title:   fmt.Sprintf("Deadline approaching: %s", title),
				Message: fmt.Sprintf("You have %d days left to complete \"%s\". Don't forget!", daysLeft, title),
				Link:    link,
			})
		}
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/notifications", h.ListNotifications)
	r.GET("/notifications/unread-count", h.UnreadCount)
	r.PUT("/notifications/:id/read", h.MarkRead)
	r.PUT("/notifications/read-all", h.MarkAllRead)
}
