package app

import (
	"context"
	"encoding/json"
	"errors"
	"fgb-lp/audit"
	database "fgb-lp/database/queries"
	"fgb-lp/functions"

	"github.com/jackc/pgx/v5/pgtype"
)

var ErrInvalidCredentials = errors.New("invalid email or password")
var ErrUserExists = errors.New("a user with that email already exists")

func (a App) Login(ctx context.Context, email, password string) (string, error) {
	user, err := a.Queries.GetUserByEmail(ctx, email)
	if err != nil {
		audit.Log(a.Queries, nil, "login_failed", map[string]any{
			"email":  email,
			"reason": "user_not_found",
		})
		return "", ErrInvalidCredentials
	}

	if !functions.ValidateHash(password, user.PasswordHash) {
		audit.Log(a.Queries, nil, "login_failed", map[string]any{
			"email":  email,
			"reason": "invalid_password",
		})
		return "", ErrInvalidCredentials
	}

	token := functions.MakeTokens()
	_, err = a.Queries.CreateSession(ctx, database.CreateSessionParams{
		UserID: user.ID,
		Token:  token,
	})
	if err != nil {
		return "", err
	}

	audit.Log(a.Queries, nil, "login", map[string]any{
		"user_id": user.ID,
		"email":   user.Email,
		"name":    user.Name,
		"role":    user.Role,
	})

	return token, nil
}

func (a App) SetupAdmin(ctx context.Context, email, password, name string) (string, error) {
	isFirst := a.CheckIfFirstUser()
	if !isFirst {
		return "", errors.New("admin user already exists")
	}

	_, err := a.Queries.GetUserByEmail(ctx, email)
	if err == nil {
		return "", ErrUserExists
	}

	hash := functions.MakeHash(password)
	user, err := a.Queries.CreateUser(ctx, database.CreateUserParams{
		Email:        email,
		PasswordHash: hash,
		Name:         name,
		Role:         "admin",
	})
	if err != nil {
		return "", err
	}

	token := functions.MakeTokens()
	_, err = a.Queries.CreateSession(ctx, database.CreateSessionParams{
		UserID: user.ID,
		Token:  token,
	})
	if err != nil {
		return "", err
	}

	audit.Log(a.Queries, nil, "admin_setup", map[string]any{
		"user_id": user.ID,
		"email":   user.Email,
		"name":    user.Name,
	})

	return token, nil
}

func (a App) Logout(ctx context.Context, token string) error {
	session, err := a.Queries.GetSessionByToken(ctx, token)
	if err == nil {
		user, uerr := a.Queries.GetUserByID(ctx, session.UserID)
		if uerr == nil {
			details, _ := json.Marshal(map[string]any{
				"user_id": user.ID,
				"email":   user.Email,
			})
			a.Queries.InsertAuditLog(ctx, database.InsertAuditLogParams{
				UserID:  pgtype.Int8{Int64: user.ID, Valid: true},
				Action:  "logout",
				Details: details,
			})
		}
	}
	return a.Queries.DeleteSession(ctx, token)
}

func (a App) ValidateSession(ctx context.Context, token string) (database.Session, error) {
	return a.Queries.GetSessionByToken(ctx, token)
}
