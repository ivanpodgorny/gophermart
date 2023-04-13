package repository

import (
	"context"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	inerr "github.com/ivanpodgorny/gophermart/internal/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUser_Create(t *testing.T) {
	var (
		ctx             = context.Background()
		id              = 1
		login           = "login"
		duplicatedLogin = "duplicatedLogin"
		hash            = "hash"
		query           = "INSERT INTO users (login, password_hash) VALUES ($1, $2) RETURNING id"
	)

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	r := NewUser(db)

	mock.ExpectQuery(query).
		WithArgs(login, hash).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id))
	mock.ExpectQuery(query).
		WithArgs(duplicatedLogin, hash).
		WillReturnError(&pgconn.PgError{Code: pgerrcode.UniqueViolation})

	insertedID, err := r.Create(ctx, login, hash)
	assert.NoError(t, err, "успешное добавление пользователя")
	assert.Equal(t, id, insertedID, "успешное добавление пользователя")

	_, err = r.Create(ctx, duplicatedLogin, hash)
	assert.ErrorIs(t, err, inerr.ErrUserExists, "добавление пользователя с существующим логином")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUser_FindByLogin(t *testing.T) {
	var (
		ctx                    = context.Background()
		id               int64 = 1
		login                  = "login"
		nonexistentLogin       = "nonexistentLogin"
		hash                   = "hash"
		query                  = "SELECT id, password_hash FROM users WHERE login = $1"
	)

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	r := NewUser(db)

	mock.ExpectQuery(query).
		WithArgs(login).
		WillReturnRows(sqlmock.NewRows([]string{"id", "password_hash"}).AddRow(id, hash))
	mock.ExpectQuery(query).
		WithArgs(nonexistentLogin).
		WillReturnError(errors.New(""))

	foundID, foundHash, err := r.FindByLogin(ctx, login)
	assert.NoError(t, err, "успешное получение данных пользователя")
	assert.Equal(t, int(id), foundID, "успешное получение данных пользователя")
	assert.Equal(t, hash, foundHash, "успешное получение данных пользователя")

	_, _, err = r.FindByLogin(ctx, nonexistentLogin)
	assert.Error(t, err, "ошибка при получении данных пользователя")

	assert.NoError(t, mock.ExpectationsWereMet())
}
