package repository

import (
	"context"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ivanpodgorny/gophermart/internal/entity"
	inerr "github.com/ivanpodgorny/gophermart/internal/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestTransaction_Create(t *testing.T) {
	var (
		ctx              = context.Background()
		userID           = 1
		order            = "148561163482734"
		amount           = 100.0
		wrongAmount      = 100.0
		tt               = entity.TransactionTypeOut
		insertOrderQuery = "INSERT INTO orders (user_id, num) VALUES ($1, $2)"
		insertTxQuery    = "INSERT INTO transactions (user_id, order_num, amount, type) VALUES ($1, $2, $3, $4)"
	)

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	r := NewTransaction(db)

	mock.ExpectBegin()
	mock.ExpectExec(insertOrderQuery).
		WithArgs(userID, order).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(insertTxQuery).
		WithArgs(userID, order, amount, tt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	mock.ExpectBegin()
	mock.ExpectExec(insertOrderQuery).
		WithArgs(userID, order).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(insertTxQuery).
		WithArgs(userID, order, wrongAmount, tt).
		WillReturnError(&pgconn.PgError{Code: pgerrcode.CheckViolation})
	mock.ExpectRollback()

	assert.NoError(
		t,
		r.Create(ctx, userID, order, amount, tt),
		"успешное создание транзакции",
	)
	assert.ErrorIs(
		t,
		r.Create(ctx, userID, order, wrongAmount, tt),
		inerr.ErrInsufficientFunds,
		"ошибка при создании транзакции",
	)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTransaction_FindAllByUserID(t *testing.T) {
	var (
		ctx          = context.Background()
		userID       = 1
		errUserID    = 2
		tt           = entity.TransactionTypeOut
		transactions = []entity.Transaction{
			{
				Order:       "148561163482734",
				Sum:         0,
				ProcessedAt: time.Now(),
			},
			{
				Order:       "267624438264306",
				Sum:         100,
				ProcessedAt: time.Now(),
			},
		}
		query = `
SELECT order_num, amount, processed_at
FROM transactions
WHERE user_id = $1
  AND type = $2
ORDER BY processed_at
`
	)

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	r := NewTransaction(db)

	rows := sqlmock.NewRows([]string{"order_num", "amount", "processed_at"})
	for _, tx := range transactions {
		rows.AddRow(tx.Order, tx.Sum, tx.ProcessedAt)
	}
	mock.ExpectQuery(query).
		WithArgs(userID, tt).
		WillReturnRows(rows)
	mock.ExpectQuery(query).
		WithArgs(errUserID, tt).
		WillReturnError(errors.New(""))

	foundTransactions, err := r.FindAllByUserID(ctx, userID, tt)
	assert.NoError(t, err, "успешное получение списаний пользователя")
	assert.Equal(t, transactions, foundTransactions, "успешное получение списаний пользователя")

	_, err = r.FindAllByUserID(ctx, errUserID, tt)
	assert.Error(t, err, "ошибка при получении списаний пользователя")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTransaction_GetBalance(t *testing.T) {
	var (
		ctx       = context.Background()
		userID    = 1
		errUserID = 2
		accrued   = 100.0
		withdrawn = 20.0
		query     = `
SELECT (SELECT coalesce(sum(amount), 0) FROM transactions WHERE user_id = $1 AND type = 'IN')  accrued,
       (SELECT coalesce(sum(amount), 0) FROM transactions WHERE user_id = $1 AND type = 'OUT') withdrawn
`
	)

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	r := NewTransaction(db)

	mock.ExpectQuery(query).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"accrued", "withdrawn"}).AddRow(accrued, withdrawn))
	mock.ExpectQuery(query).
		WithArgs(errUserID).
		WillReturnError(errors.New(""))

	foundCurrent, foundWithdrawn, err := r.GetBalance(ctx, userID)
	assert.NoError(t, err, "успешное получение баланса пользователя")
	assert.Equal(t, accrued-withdrawn, foundCurrent, "успешное получение баланса пользователя")
	assert.Equal(t, withdrawn, foundWithdrawn, "успешное получение баланса пользователя")

	_, _, err = r.GetBalance(ctx, errUserID)
	assert.Error(t, err, "ошибка при получении баланса пользователя")

	assert.NoError(t, mock.ExpectationsWereMet())
}
