package app

import (
	"context"
	"errors"
	database "fgb-lp/database/queries"
	"fgb-lp/functions"
)

var ErrInvalidCredentials = errors.New("invalid email or password")
var ErrUserExists = errors.New("a user with that email already exists")

func (a App) Login(ctx context.Context, email, password string) (string, error) {
	user, err := a.Queries.GetUserByEmail(ctx, email)
	if err != nil {
		return "", ErrInvalidCredentials
	}

	if !functions.ValidateHash(password, user.PasswordHash) {
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

	return token, nil
}

func (a App) Logout(ctx context.Context, token string) error {
	return a.Queries.DeleteSession(ctx, token)
}

func (a App) ValidateSession(ctx context.Context, token string) (database.Session, error) {
	return a.Queries.GetSessionByToken(ctx, token)
}
