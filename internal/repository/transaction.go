package repository

import (
	"context"
	"database/sql"
	"github.com/ivanpodgorny/gophermart/internal/entity"
	inerr "github.com/ivanpodgorny/gophermart/internal/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type Transaction struct {
	db *sql.DB
}

func NewTransaction(db *sql.DB) *Transaction {
	return &Transaction{db: db}
}

// GetBalance возвращает сумму доступных и списанных баллов пользователя.
func (r *Transaction) GetBalance(ctx context.Context, userID int) (float64, float64, error) {
	var accrued, withdrawn float64
	err := r.db.QueryRowContext(ctx, `
SELECT (SELECT coalesce(sum(amount), 0) FROM transactions WHERE user_id = $1 AND type = 'IN')  accrued,
       (SELECT coalesce(sum(amount), 0) FROM transactions WHERE user_id = $1 AND type = 'OUT') withdrawn
	`, userID).Scan(&accrued, &withdrawn)

	return accrued - withdrawn, withdrawn, err
}

// Create создает запись о списании или начислении баллов для пользователя. При попытке списать
// недоступную сумму возвращает ошибку errors.ErrInsufficientFunds.
func (r *Transaction) Create(ctx context.Context, userID int, order string, sum float64, t entity.TransactionType) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx, "INSERT INTO orders (user_id, num) VALUES ($1, $2)", userID, order); err != nil {
		_ = tx.Rollback()

		return err
	}

	_, err = tx.ExecContext(
		ctx,
		"INSERT INTO transactions (user_id, order_num, amount, type) VALUES ($1, $2, $3, $4)",
		userID,
		order,
		sum,
		t,
	)
	if err != nil {
		_ = tx.Rollback()
		if err.(*pgconn.PgError).Code == pgerrcode.CheckViolation {
			err = inerr.ErrInsufficientFunds
		}

		return err
	}

	if err = tx.Commit(); err != nil {
		_ = tx.Rollback()

		return err
	}

	return nil
}

// FindAllByUserID возвращает список транзакций пользователя типа t. Данные отсортированы
// по времени транзакции от самых старых к самым новым.
func (r *Transaction) FindAllByUserID(ctx context.Context, userID int, t entity.TransactionType) (txs []entity.Transaction, err error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT order_num, amount, processed_at
FROM transactions
WHERE user_id = $1
  AND type = $2
ORDER BY processed_at
	`, userID, t)
	if err != nil {
		return nil, err
	}

	defer func(rows *sql.Rows) {
		err = rows.Close()
	}(rows)

	for rows.Next() {
		tx := entity.Transaction{}
		err = rows.Scan(&tx.Order, &tx.Sum, &tx.ProcessedAt)
		if err != nil {
			continue
		}

		txs = append(txs, tx)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return txs, err
}
