package service

import (
	"context"
	"errors"
	"github.com/ivanpodgorny/gophermart/internal/entity"
	inerr "github.com/ivanpodgorny/gophermart/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

type OrderRepositoryMock struct {
	mock.Mock
}

func (m *OrderRepositoryMock) Create(_ context.Context, userID int, num string) error {
	args := m.Called(userID, num)

	return args.Error(0)
}

func (m *OrderRepositoryMock) FindAllByUserID(_ context.Context, userID int) ([]entity.Order, error) {
	args := m.Called(userID)

	return args.Get(0).([]entity.Order), args.Error(1)
}

func TestOrder_Create(t *testing.T) {
	var (
		ctx           = context.Background()
		userID        = 1
		num           = "166221614883769"
		duplicatedNum = "267624438264306"
		repository    = &OrderRepositoryMock{}
		queue         = make(chan entity.StatusCheckJob, 1)
	)

	defer close(queue)

	repository.
		On("Create", userID, num).
		Return(nil).
		Once()
	repository.
		On("Create", userID, duplicatedNum).
		Return(inerr.ErrOrderExists).
		Once()
	service := Order{
		repository: repository,
		queue:      queue,
	}

	assert.NoError(
		t,
		service.Create(ctx, userID, num),
		"успешное добавление заказа",
	)
	assert.Equal(
		t,
		entity.NewStatusCheckJob(num),
		<-queue,
		"успешное добавление задачи для проверки статуса начисления",
	)

	assert.ErrorIs(
		t,
		service.Create(ctx, userID, duplicatedNum),
		inerr.ErrOrderExists,
		"ошибка при добавлении заказа",
	)
	assert.Never(
		t,
		func() bool { return len(queue) > 0 },
		100*time.Millisecond,
		20*time.Millisecond,
		"задача для проверки статуса начисления не создается при ошибке",
	)

	repository.AssertExpectations(t)
}

func TestOrder_GetAll(t *testing.T) {
	var (
		ctx         = context.Background()
		userID      = 1
		errorUserID = 2
		orders      = []entity.Order{
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
		repository = &OrderRepositoryMock{}
	)
	repository.
		On("FindAllByUserID", userID).
		Return(orders, nil).
		Once()
	repository.
		On("FindAllByUserID", errorUserID).
		Return([]entity.Order{}, errors.New("")).
		Once()
	service := Order{repository: repository}

	resOrders, _ := service.GetAll(ctx, userID)
	assert.Equal(t, orders, resOrders, "успешное получение списка заказов")

	_, err := service.GetAll(ctx, errorUserID)
	assert.Error(t, err, "ошибка при получении списка заказов")

	repository.AssertExpectations(t)
}
