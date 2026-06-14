package ai

import (
	"context"
	database "fgb-lp/database/queries"

	"github.com/jackc/pgx/v5/pgtype"
)

// notifyAdmins sends a notification to every admin user.
func notifyAdmins(queries *database.Queries, title, message, link string) {
	admins, err := queries.GetAdminUsers(context.Background())
	if err != nil {
		return
	}
	for _, admin := range admins {
		queries.CreateNotification(context.Background(), database.CreateNotificationParams{
			UserID:  pgtype.Int8{Int64: admin.ID, Valid: true},
			Title:   title,
			Message: message,
			Link:    link,
		})
	}
}

// notifyAll sends a notification to every user.
func notifyAll(queries *database.Queries, title, message, link string) {
	users, err := queries.GetAllUsers(context.Background())
	if err != nil {
		return
	}
	for _, user := range users {
		queries.CreateNotification(context.Background(), database.CreateNotificationParams{
			UserID:  pgtype.Int8{Int64: user.ID, Valid: true},
			Title:   title,
			Message: message,
			Link:    link,
		})
	}
}
