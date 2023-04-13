package repository

import (
	"context"
	"database/sql"
)

type Token struct {
	db *sql.DB
}

func NewToken(db *sql.DB) *Token {
	return &Token{db: db}
}

// Save сохраняет авторизационный токен пользователя.
func (r *Token) Save(ctx context.Context, token string, userID int) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO tokens (token, user_id) VALUES ($1, $2)", token, userID)

	return err
}

// FindUserID возвращает идентификатор пользователя для данного токена.
func (r *Token) FindUserID(ctx context.Context, token string) (int, error) {
	userID := 0
	err := r.db.QueryRowContext(ctx, "SELECT user_id FROM tokens WHERE token = $1", token).Scan(&userID)

	return userID, err
}
