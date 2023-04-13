package worker

import (
	"context"
	"github.com/ivanpodgorny/gophermart/internal/entity"
	"log"
	"sync"
)

// StatusChecker проверяет статус начисления в системе расчёта начислений баллов лояльности
// и создает задачу на обновление заказа, если статус обновился. Для выполнения запросов
// на проверку создается StatusChecker.workersCount воркеров. При вызове NewStatusChecker
// добавляет в очередь на проверку сохраненные необработанные заказы.
type StatusChecker struct {
	repository   CheckerRepository
	client       AccrualClient
	jobs         chan entity.StatusCheckJob
	results      chan<- entity.StatusCheckResult
	wg           *sync.WaitGroup
	workersCount int
}

type CheckerRepository interface {
	FindUnprocessed(ctx context.Context) []entity.Order
}

type AccrualClient interface {
	GetAccrual(ctx context.Context, order string) (status entity.OrderStatus, accrual float64, err error)
}

func NewStatusChecker(
	ctx context.Context,
	r CheckerRepository,
	c AccrualClient,
	j chan entity.StatusCheckJob,
	res chan<- entity.StatusCheckResult,
	wg *sync.WaitGroup,
	w int,
) *StatusChecker {
	checker := &StatusChecker{
		repository:   r,
		client:       c,
		jobs:         j,
		results:      res,
		wg:           wg,
		workersCount: w,
	}

	for _, o := range checker.repository.FindUnprocessed(ctx) {
		go func(order entity.Order) {
			checker.jobs <- entity.StatusCheckJob{
				Num:    order.Number,
				Status: order.Status,
			}
		}(o)
	}

	return checker
}

func (c *StatusChecker) Do(ctx context.Context) {
	for i := 0; i < c.workersCount; i++ {
		c.wg.Add(1)

		go c.worker(ctx)
	}
}

func (c *StatusChecker) worker(ctx context.Context) {
	defer c.wg.Done()

	for {
		select {
		case j, ok := <-c.jobs:
			if !ok {
				return
			}

			status, accrual, err := c.client.GetAccrual(ctx, j.Num)
			if err != nil {
				c.jobs <- j
				log.Printf("ошибка получения статуса заказа %s: %v", j.Num, err)

				continue
			}

			if status != j.Status {
				j.Status = status
				c.results <- entity.StatusCheckResult{
					Num:     j.Num,
					Status:  status,
					Accrual: accrual,
				}
			}
			if status != entity.OrderStatusInvalid && status != entity.OrderStatusProcessed {
				c.jobs <- j
			}
		case <-ctx.Done():
			return
		}
	}
}
