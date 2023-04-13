package worker

import (
	"context"
	"github.com/ivanpodgorny/gophermart/internal/entity"
	"log"
	"sync"
)

// OrderUpdater получает задачи на обновление статусов заказов и выполняет обновление.
// Для выполнения обновлений создается OrderUpdater.workersCount воркеров.
type OrderUpdater struct {
	repository   UpdaterRepository
	queue        <-chan entity.StatusCheckResult
	wg           *sync.WaitGroup
	workersCount int
}

type UpdaterRepository interface {
	UpdateStatus(ctx context.Context, num string, status entity.OrderStatus, accrual float64) error
}

func NewOrderUpdater(r UpdaterRepository, q <-chan entity.StatusCheckResult, wg *sync.WaitGroup, w int) *OrderUpdater {
	return &OrderUpdater{
		repository:   r,
		queue:        q,
		wg:           wg,
		workersCount: w,
	}
}

func (u *OrderUpdater) Do(ctx context.Context) {
	for i := 0; i < u.workersCount; i++ {
		u.wg.Add(1)

		go u.worker(ctx)
	}
}

func (u *OrderUpdater) worker(ctx context.Context) {
	defer u.wg.Done()

	for {
		select {
		case res, ok := <-u.queue:
			if !ok {
				return
			}

			if err := u.repository.UpdateStatus(ctx, res.Num, res.Status, res.Accrual); err != nil {
				log.Printf("ошибка обновления статуса заказа %s: %v", res.Num, err)
			}
		case <-ctx.Done():
			return
		}
	}
}
