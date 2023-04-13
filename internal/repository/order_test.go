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

func TestOrder_Create(t *testing.T) {
	var (
		ctx              = context.Background()
		userID           = 1
		anotherUserID    = 2
		order            = "148561163482734"
		duplicatedOrder  = "267624438264306"
		anotherUserOrder = "166221614883769"
		insertQuery      = "INSERT INTO orders (user_id, num, status) VALUES ($1, $2, 'NEW')"
		getUserQuery     = "SELECT user_id FROM orders WHERE num = $1"
	)

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	r := NewOrder(db)

	mock.ExpectExec(insertQuery).
		WithArgs(userID, order).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(insertQuery).
		WithArgs(userID, duplicatedOrder).
		WillReturnError(&pgconn.PgError{Code: pgerrcode.UniqueViolation})
	mock.ExpectQuery(getUserQuery).
		WithArgs(duplicatedOrder).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(userID))
	mock.ExpectExec(insertQuery).
		WithArgs(userID, anotherUserOrder).
		WillReturnError(&pgconn.PgError{Code: pgerrcode.UniqueViolation})
	mock.ExpectQuery(getUserQuery).
		WithArgs(anotherUserOrder).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(anotherUserID))

	assert.NoError(t, r.Create(ctx, userID, order), "успешное добавление заказа")
	assert.ErrorIs(
		t,
		r.Create(ctx, userID, duplicatedOrder),
		inerr.ErrOrderExists,
		"попытка добавить добавленный ранее заказ",
	)
	assert.ErrorIs(
		t,
		r.Create(ctx, userID, anotherUserOrder),
		inerr.ErrOrderNotBelongToUser,
		"попытка добавить заказ, добавленный другим пользователем",
	)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrder_FindAllByUserID(t *testing.T) {
	var (
		ctx       = context.Background()
		userID    = 1
		errUserID = 2
		orders    = []entity.Order{
			{
				Number:     "148561163482734",
				Status:     entity.OrderStatusProcessing,
				Accrual:    0,
				UploadedAt: time.Now(),
			},
			{
				Number:     "267624438264306",
				Status:     entity.OrderStatusProcessed,
				Accrual:    100,
				UploadedAt: time.Now(),
			},
		}
		query = `
SELECT num, status, accrual, uploaded_at
FROM orders
WHERE user_id = $1
  AND status IS NOT NULL
ORDER BY uploaded_at
`
	)

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	r := NewOrder(db)

	rows := sqlmock.NewRows([]string{"num", "status", "accrual", "uploaded_at"})
	for _, o := range orders {
		rows.AddRow(o.Number, o.Status, o.Accrual, o.UploadedAt)
	}
	mock.ExpectQuery(query).
		WithArgs(userID).
		WillReturnRows(rows)
	mock.ExpectQuery(query).
		WithArgs(errUserID).
		WillReturnError(errors.New(""))

	foundOrders, err := r.FindAllByUserID(ctx, userID)
	assert.NoError(t, err, "успешное получение заказов пользователя")
	assert.Equal(t, orders, foundOrders, "успешное получение заказов пользователя")

	_, err = r.FindAllByUserID(ctx, errUserID)
	assert.Error(t, err, "ошибка при получении заказов пользователя")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrder_FindUnprocessed(t *testing.T) {
	var (
		ctx    = context.Background()
		orders = []entity.Order{
			{
				Number: "148561163482734",
				Status: entity.OrderStatusProcessing,
			},
			{
				Number: "267624438264306",
				Status: entity.OrderStatusNew,
			},
		}
	)

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	r := NewOrder(db)

	rows := sqlmock.NewRows([]string{"num", "status"})
	for _, o := range orders {
		rows.AddRow(o.Number, o.Status)
	}
	mock.
		ExpectQuery("SELECT num, status FROM orders WHERE status IN ('NEW', 'PROCESSING')").
		WillReturnRows(rows)

	assert.Equal(t, orders, r.FindUnprocessed(ctx), "успешное получение необработанных заказов")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrder_UpdateStatus(t *testing.T) {
	var (
		ctx              = context.Background()
		userID           = 1
		unprocessedOrder = entity.Order{
			Number:  "148561163482734",
			Status:  entity.OrderStatusProcessing,
			Accrual: 0,
		}
		processedOrder = entity.Order{
			Number:  "267624438264306",
			Status:  entity.OrderStatusProcessed,
			Accrual: 100,
		}
		processedOrderError = entity.Order{
			Number:  "267624438264306",
			Status:  entity.OrderStatusProcessed,
			Accrual: 100,
		}
		updateQuery = "UPDATE orders SET status = $1, accrual = $2 WHERE num = $3 RETURNING user_id"
		insertQuery = "INSERT INTO transactions (order_num, amount, type, user_id) VALUES ($1, $2, 'IN', $3)"
	)

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	r := NewOrder(db)

	mock.ExpectBegin()
	mock.
		ExpectQuery(updateQuery).
		WithArgs(unprocessedOrder.Status, unprocessedOrder.Accrual, unprocessedOrder.Number).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(userID))
	mock.ExpectCommit()

	mock.ExpectBegin()
	mock.
		ExpectQuery(updateQuery).
		WithArgs(processedOrder.Status, processedOrder.Accrual, processedOrder.Number).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(userID))
	mock.
		ExpectExec(insertQuery).
		WithArgs(processedOrder.Number, processedOrder.Accrual, userID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	mock.ExpectBegin()
	mock.
		ExpectQuery(updateQuery).
		WithArgs(processedOrderError.Status, processedOrderError.Accrual, processedOrderError.Number).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(userID))
	mock.
		ExpectExec(insertQuery).
		WithArgs(processedOrderError.Number, processedOrderError.Accrual, userID).
		WillReturnError(errors.New(""))
	mock.ExpectRollback()

	assert.NoError(
		t,
		r.UpdateStatus(ctx, unprocessedOrder.Number, unprocessedOrder.Status, unprocessedOrder.Accrual),
		"успешное обновление необработанного заказа",
	)
	assert.NoError(
		t,
		r.UpdateStatus(ctx, processedOrder.Number, processedOrder.Status, processedOrder.Accrual),
		"успешное обновление обработанного заказа",
	)
	assert.Error(
		t,
		r.UpdateStatus(ctx, processedOrderError.Number, processedOrderError.Status, processedOrderError.Accrual),
		"ошибка при обновлении обработанного заказа",
	)
	assert.NoError(t, mock.ExpectationsWereMet())
}
