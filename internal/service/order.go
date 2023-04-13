package service

import (
	"context"
	"github.com/ivanpodgorny/gophermart/internal/entity"
)

type Order struct {
	repository OrderRepository
	queue      chan<- entity.StatusCheckJob
}

type OrderRepository interface {
	Create(ctx context.Context, userID int, num string) error
	FindAllByUserID(ctx context.Context, userID int) ([]entity.Order, error)
}

func NewOrder(r OrderRepository, q chan<- entity.StatusCheckJob) *Order {
	return &Order{
		repository: r,
		queue:      q,
	}
}

// Create добавляет новый заказ и создает задачу на проверку статуса начисления по нему.
func (s *Order) Create(ctx context.Context, userID int, num string) error {
	if err := s.repository.Create(ctx, userID, num); err != nil {
		return err
	}

	go func() {
		s.queue <- entity.NewStatusCheckJob(num)
	}()

	return nil
}

// GetAll возвращает список добавленных заказов пользователя.
func (s *Order) GetAll(ctx context.Context, userID int) ([]entity.Order, error) {
	return s.repository.FindAllByUserID(ctx, userID)
}
