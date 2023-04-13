package service

import (
	"context"
	"github.com/ivanpodgorny/gophermart/internal/entity"
)

type Transaction struct {
	repository TransactionRepository
}

type TransactionRepository interface {
	GetBalance(ctx context.Context, userID int) (current float64, withdrawn float64, err error)
	Create(ctx context.Context, userID int, order string, sum float64, t entity.TransactionType) error
	FindAllByUserID(ctx context.Context, userID int, t entity.TransactionType) ([]entity.Transaction, error)
}

func NewTransaction(r TransactionRepository) *Transaction {
	return &Transaction{repository: r}
}

// GetBalance возвращает данные о доступных и списанных баллах пользователя.
func (s *Transaction) GetBalance(ctx context.Context, userID int) (float64, float64, error) {
	return s.repository.GetBalance(ctx, userID)
}

// Withdraw создает списание баллов в счёт оплаты заказа.
func (s *Transaction) Withdraw(ctx context.Context, userID int, order string, sum float64) error {
	return s.repository.Create(ctx, userID, order, sum, entity.TransactionTypeOut)
}

// GetWithdrawals возвращает список всех списаний пользователя.
func (s *Transaction) GetWithdrawals(ctx context.Context, userID int) ([]entity.Transaction, error) {
	return s.repository.FindAllByUserID(ctx, userID, entity.TransactionTypeOut)
}
