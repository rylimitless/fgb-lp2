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

// dummyHash is a precomputed Argon2id hash used to equalize login timing when
// the email is not found. Without it, user-not-found returns immediately
// while invalid-password runs a full hash compare, enabling email enumeration
// via timing (fixes audit item H5). It is a valid hash of a random password
// that nobody knows.
var dummyHash = functions.MakeHash("timing-equalization-placeholder")

func (a App) Login(ctx context.Context, email, password string) (string, error) {
	user, err := a.Queries.GetUserByEmail(ctx, email)
	if err != nil {
		// Run a dummy hash validation so the not-found path takes roughly the
		// same time as the invalid-password path.
		_ = functions.ValidateHash(password, dummyHash)
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

	token, err := functions.MakeTokens()
	if err != nil {
		return "", err
	}
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
	// Serialize first-admin setup with a transaction-scoped advisory lock so two
	// concurrent /api/setup requests can't both pass the first-user check and
	// create two admins (fixes audit item H6). The lock is released on commit/
	// rollback, and the check+insert run inside the same tx so they're atomic.
	tx, err := a.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	// pg_advisory_xact_lock takes a 32-bit key; hashtext('setup') gives a stable one.
	if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock(hashtext('setup_admin'))`); err != nil {
		return "", err
	}

	txQueries := a.Queries.WithTx(tx)

	count, err := txQueries.CheckIfFirstUser(ctx)
	if err != nil {
		return "", err
	}
	if count != 0 {
		return "", errors.New("admin user already exists")
	}

	_, err = txQueries.GetUserByEmail(ctx, email)
	if err == nil {
		return "", ErrUserExists
	}

	hash := functions.MakeHash(password)
	user, err := txQueries.CreateUser(ctx, database.CreateUserParams{
		Email:        email,
		PasswordHash: hash,
		Name:         name,
		Role:         "admin",
	})
	if err != nil {
		return "", err
	}

	token, err := functions.MakeTokens()
	if err != nil {
		return "", err
	}
	_, err = txQueries.CreateSession(ctx, database.CreateSessionParams{
		UserID: user.ID,
		Token:  token,
	})
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
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
