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

type CheckerRepositoryMock struct {
	mock.Mock
}

func (m *CheckerRepositoryMock) FindUnprocessed(_ context.Context) []entity.Order {
	args := m.Called()

	return args.Get(0).([]entity.Order)
}

type AccrualClientMock struct {
	mock.Mock
}

func (m *AccrualClientMock) GetAccrual(_ context.Context, order string) (entity.OrderStatus, float64, error) {
	args := m.Called(order)

	return args.Get(0).(entity.OrderStatus), args.Get(1).(float64), args.Error(2)
}

func TestNewStatusChecker(t *testing.T) {
	var (
		orders = []entity.Order{
			{
				Number: "148561163482734",
				Status: entity.OrderStatusProcessing,
			},
			{
				Number: "267624438264306",
				Status: entity.OrderStatusProcessed,
			},
		}
		jobs = []entity.StatusCheckJob{
			{
				Num:    "148561163482734",
				Status: entity.OrderStatusProcessing,
			},
			{
				Num:    "267624438264306",
				Status: entity.OrderStatusProcessed,
			},
		}
		jobsCh     = make(chan entity.StatusCheckJob, 4)
		repository = &CheckerRepositoryMock{}
	)
	repository.On("FindUnprocessed").Return(orders).Once()
	NewStatusChecker(
		context.Background(),
		repository,
		&AccrualClientMock{},
		jobsCh,
		make(chan entity.StatusCheckResult, 4),
		&sync.WaitGroup{},
		4,
	)

	for i := 0; i < len(orders); i++ {
		assert.Contains(t, jobs, <-jobsCh, "успешная загрузка необработанных заказов")
	}

	repository.AssertExpectations(t)
}

func TestStatusChecker_Do(t *testing.T) {
	var (
		ctx, cancel = context.WithCancel(context.Background())
		client      = &AccrualClientMock{}
		jobsCh      = make(chan entity.StatusCheckJob, 4)
		resultsCh   = make(chan entity.StatusCheckResult, 4)
		jobs        = []entity.StatusCheckJob{
			{
				Num:    "711388585544181",
				Status: entity.OrderStatusNew,
			},
			{
				Num:    "655770442208670",
				Status: entity.OrderStatusProcessing,
			},
			{
				Num:    "116322550058324",
				Status: entity.OrderStatusProcessing,
			},
			{
				Num:    "116322550058324",
				Status: entity.OrderStatusNew,
			},
		}
		results = []entity.StatusCheckResult{
			{
				Num:     "711388585544181",
				Status:  entity.OrderStatusProcessed,
				Accrual: 50,
			},
			{
				Num:     "655770442208670",
				Status:  entity.OrderStatusInvalid,
				Accrual: 50,
			},
			{
				Num:     "116322550058324",
				Status:  entity.OrderStatusProcessed,
				Accrual: 100,
			},
			{
				Num:     "116322550058324",
				Status:  entity.OrderStatusProcessed,
				Accrual: 100,
			},
		}
	)

	defer close(jobsCh)
	defer close(resultsCh)

	for i := range jobs {
		j := jobs[i]
		r := results[i]
		jobsCh <- j
		client.On("GetAccrual", j.Num).Return(r.Status, r.Accrual, nil).Once()
	}
	checker := StatusChecker{
		client:       client,
		jobs:         jobsCh,
		results:      resultsCh,
		wg:           &sync.WaitGroup{},
		workersCount: 4,
	}

	checker.Do(ctx)

	assert.Eventually(
		t,
		func() bool { return len(jobsCh) == 0 },
		100*time.Millisecond,
		10*time.Millisecond,
		"успешная обработка очереди",
	)
	for i := 0; i < len(results); i++ {
		assert.Contains(t, results, <-resultsCh, "успешное создание задач на обновление заказов")
	}

	cancel()
	for _, j := range jobs {
		jobsCh <- j
	}
	assert.Eventually(
		t,
		func() bool { return len(jobsCh) == 4 },
		100*time.Millisecond,
		10*time.Millisecond,
		"корректное завершение работы при отмене контекста",
	)

	client.AssertExpectations(t)
}
