package app

import (
	"context"
	database "fgb-lp/database/queries"

	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	Pool    *pgxpool.Pool
	Queries *database.Queries
}

func (a App) CheckIfFirstUser() bool {
	count, err := a.Queries.CheckIfFirstUser(context.Background())
	if err != nil {
		return false
	}
	return count == 0
}

