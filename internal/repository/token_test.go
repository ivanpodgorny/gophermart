package repository

import (
	"context"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestToken_Save(t *testing.T) {
	var (
		ctx         = context.Background()
		userID      = 1
		wrongUserID = 2
		token       = "token"
		query       = "INSERT INTO tokens (token, user_id) VALUES ($1, $2)"
	)

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	r := NewToken(db)

	mock.ExpectExec(query).
		WithArgs(token, userID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(query).
		WithArgs(token, wrongUserID).
		WillReturnError(errors.New(""))

	assert.NoError(
		t,
		r.Save(ctx, token, userID),
		"успешное сохранение токена",
	)
	assert.Error(
		t,
		r.Save(ctx, token, wrongUserID),
		"ошибка при сохранении токена",
	)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestToken_FindUserID(t *testing.T) {
	var (
		ctx              = context.Background()
		userID           = 1
		token            = "token"
		nonexistentToken = "nonexistentToken"
		query            = "SELECT user_id FROM tokens WHERE token = $1"
	)

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	r := NewToken(db)

	mock.ExpectQuery(query).
		WithArgs(token).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(userID))
	mock.ExpectQuery(query).
		WithArgs(nonexistentToken).
		WillReturnError(errors.New(""))

	foundID, err := r.FindUserID(ctx, token)
	assert.NoError(t, err, "успешное получение id пользователя")
	assert.Equal(t, userID, foundID, "успешное получение id пользователя")

	_, err = r.FindUserID(ctx, nonexistentToken)
	assert.Error(t, err, "ошибка при получении id пользователя")

	assert.NoError(t, mock.ExpectationsWereMet())
}
