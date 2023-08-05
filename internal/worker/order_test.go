package worker

import (
	"context"
	"github.com/ivanpodgorny/gophermart/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sync"
	"testing"
	"time"
)

type UpdaterRepositoryMock struct {
	mock.Mock
}

func (m *UpdaterRepositoryMock) UpdateStatus(_ context.Context, n string, s entity.OrderStatus, a float64) error {
	args := m.Called(n, s, a)

	return args.Error(0)
}

func TestOrderUpdater_Do(t *testing.T) {
	var (
		ctx, cancel = context.WithCancel(context.Background())
		repository  = &UpdaterRepositoryMock{}
		queue       = make(chan entity.StatusCheckResult, 4)
		jobs        = []entity.StatusCheckResult{
			{
				Num:     "711388585544181",
				Status:  entity.OrderStatusProcessed,
				Accrual: 50,
			},
			{
				Num:     "655770442208670",
				Status:  entity.OrderStatusInvalid,
				Accrual: 0,
			},
			{
				Num:     "116322550058324",
				Status:  entity.OrderStatusProcessing,
				Accrual: 0,
			},
			{
				Num:     "116322550058324",
				Status:  entity.OrderStatusProcessed,
				Accrual: 100,
			},
		}
	)

	defer close(queue)

	for i := range jobs {
		j := jobs[i]
		queue <- j
		repository.On("UpdateStatus", j.Num, j.Status, j.Accrual).Return(nil).Once()
	}
	updater := OrderUpdater{
		repository:   repository,
		queue:        queue,
		wg:           &sync.WaitGroup{},
		workersCount: 4,
	}

	updater.Do(ctx)

	assert.Eventually(
		t,
		func() bool { return len(queue) == 0 },
		100*time.Millisecond,
		10*time.Millisecond,
		"успешная обработка очереди",
	)

	cancel()
	for _, j := range jobs {
		queue <- j
	}
	assert.Eventually(
		t,
		func() bool { return len(queue) == 4 },
		100*time.Millisecond,
		10*time.Millisecond,
		"корректное завершение работы при отмене контекста",
	)

	repository.AssertExpectations(t)
}
