package repository

import (
	"context"
	"database/sql"
	inerr "github.com/ivanpodgorny/gophermart/internal/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type User struct {
	db *sql.DB
}

func NewUser(db *sql.DB) *User {
	return &User{db: db}
}

// Create создает нового пользователя и возвращает его id. Если пользователь с
// переданным login существует, возвращает ошибку errors.ErrUserExists.
func (r *User) Create(ctx context.Context, login, passwordHash string) (int, error) {
	id := 0
	err := r.db.QueryRowContext(
		ctx,
		"INSERT INTO users (login, password_hash) VALUES ($1, $2) RETURNING id",
		login,
		passwordHash,
	).Scan(&id)
	if err != nil && err.(*pgconn.PgError).Code == pgerrcode.UniqueViolation {
		return 0, inerr.ErrUserExists
	}

	return id, err
}

// FindByLogin возвращает id и хэш пароля пользователя с переданным login.
func (r *User) FindByLogin(ctx context.Context, login string) (int, string, error) {
	var (
		id   = 0
		hash = ""
	)
	err := r.db.QueryRowContext(ctx, "SELECT id, password_hash FROM users WHERE login = $1", login).Scan(&id, &hash)

	return id, hash, err
}
