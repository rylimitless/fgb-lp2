package notifications

import (
	"context"
	database "fgb-lp/database/queries"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	Queries *database.Queries
}

func NewHandler(queries *database.Queries) *Handler {
	return &Handler{Queries: queries}
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

// MarkRead marks a single notification as read.
func (h *Handler) MarkRead(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	_, err = h.Queries.MarkNotificationRead(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark as read"})
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

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/notifications", h.ListNotifications)
	r.GET("/notifications/unread-count", h.UnreadCount)
	r.PUT("/notifications/:id/read", h.MarkRead)
}
