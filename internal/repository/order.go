package repository

import (
	"context"
	"database/sql"
	"github.com/ivanpodgorny/gophermart/internal/entity"
	inerr "github.com/ivanpodgorny/gophermart/internal/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type Order struct {
	db *sql.DB
}

func NewOrder(db *sql.DB) *Order {
	return &Order{db: db}
}

// Create добавляет новый заказ. Если номер заказа уже был загружен этим пользователем,
// возвращает ошибку errors.ErrOrderExists. Если номер заказа уже был загружен
// другим пользователем, возвращает ошибку errors.ErrOrderNotBelongToUser.
func (r *Order) Create(ctx context.Context, userID int, num string) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO orders (user_id, num, status) VALUES ($1, $2, 'NEW')", userID, num)
	if err != nil && err.(*pgconn.PgError).Code == pgerrcode.UniqueViolation {
		ownerID := 0
		if err = r.db.QueryRowContext(ctx, "SELECT user_id FROM orders WHERE num = $1", num).Scan(&ownerID); err != nil {
			return err
		}

		err = inerr.ErrOrderExists
		if ownerID != userID {
			err = inerr.ErrOrderNotBelongToUser
		}
	}

	return err
}

// FindAllByUserID возвращает список добавленных заказов пользователя. Данные отсортированы
// по времени добавления от самых старых к самым новым.
func (r *Order) FindAllByUserID(ctx context.Context, userID int) (orders []entity.Order, err error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT num, status, accrual, uploaded_at
FROM orders
WHERE user_id = $1
  AND status IS NOT NULL
ORDER BY uploaded_at
	`, userID)
	if err != nil {
		return nil, err
	}

	defer func(rows *sql.Rows) {
		err = rows.Close()
	}(rows)

	for rows.Next() {
		order := entity.Order{}
		err = rows.Scan(&order.Number, &order.Status, &order.Accrual, &order.UploadedAt)
		if err != nil {
			continue
		}

		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return orders, err
}

// UpdateStatus обновляет статус заказа. Если статус изменился на entity.OrderStatusProcessed,
// создает транзакцию с суммой accrual.
func (r *Order) UpdateStatus(ctx context.Context, num string, status entity.OrderStatus, accrual float64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	userID := 0
	if err = tx.QueryRowContext(ctx, "UPDATE orders SET status = $1, accrual = $2 WHERE num = $3 RETURNING user_id", status, accrual, num).Scan(&userID); err != nil {
		_ = tx.Rollback()

		return err
	}

	if status == entity.OrderStatusProcessed {
		if _, err = tx.ExecContext(ctx, "INSERT INTO transactions (order_num, amount, type, user_id) VALUES ($1, $2, 'IN', $3)", num, accrual, userID); err != nil {
			_ = tx.Rollback()

			return err
		}
	}

	if err = tx.Commit(); err != nil {
		_ = tx.Rollback()

		return err
	}

	return nil
}

// FindUnprocessed возвращает список всех заказов, которые необходимо обработать
// (статус равен entity.OrderStatusNew или entity.OrderStatusProcessing).
func (r *Order) FindUnprocessed(ctx context.Context) (orders []entity.Order) {
	rows, err := r.db.QueryContext(ctx, "SELECT num, status FROM orders WHERE status IN ('NEW', 'PROCESSING')")
	if err != nil {
		return nil
	}

	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	for rows.Next() {
		order := entity.Order{}
		err = rows.Scan(&order.Number, &order.Status)
		if err != nil {
			continue
		}

		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil
	}

	return orders
}
