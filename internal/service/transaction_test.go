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

type TransactionRepositoryMock struct {
	mock.Mock
}

func (m *TransactionRepositoryMock) GetBalance(_ context.Context, userID int) (float64, float64, error) {
	args := m.Called(userID)

	return args.Get(0).(float64), args.Get(1).(float64), args.Error(2)
}

func (m *TransactionRepositoryMock) Create(_ context.Context, userID int, order string, sum float64, t entity.TransactionType) error {
	args := m.Called(userID, order, sum, t)

	return args.Error(0)
}

func (m *TransactionRepositoryMock) FindAllByUserID(_ context.Context, userID int, t entity.TransactionType) ([]entity.Transaction, error) {
	args := m.Called(userID, t)

	return args.Get(0).([]entity.Transaction), args.Error(1)
}

func TestTransaction_GetBalance(t *testing.T) {
	var (
		ctx         = context.Background()
		userID      = 1
		wrongUserID = 2
		current     = 500.5
		withdrawn   = 42.0
		repository  = &TransactionRepositoryMock{}
	)
	repository.On("GetBalance", userID).Return(current, withdrawn, nil).Once()
	repository.On("GetBalance", wrongUserID).Return(0.0, 0.0, errors.New("")).Once()
	service := Transaction{repository: repository}

	resCurrent, resWithdrawn, _ := service.GetBalance(ctx, userID)
	assert.Equal(t, current, resCurrent, "успешное получение текущей суммы баллов")
	assert.Equal(t, withdrawn, resWithdrawn, "успешное получение суммы использованных баллов")

	_, _, err := service.GetBalance(ctx, wrongUserID)
	assert.Error(t, err, "ошибка при получении баланса пользователя")

	repository.AssertExpectations(t)
}

func TestTransaction_Withdraw(t *testing.T) {
	var (
		ctx            = context.Background()
		userID         = 1
		order          = "166221614883769"
		sum            = 100.0
		unavailableSum = 1000.0
		repository     = &TransactionRepositoryMock{}
	)
	repository.
		On("Create", userID, order, sum, entity.TransactionTypeOut).
		Return(nil).
		Once()
	repository.
		On("Create", userID, order, unavailableSum, entity.TransactionTypeOut).
		Return(inerr.ErrInsufficientFunds).
		Once()
	service := Transaction{repository: repository}

	assert.NoError(
		t,
		service.Withdraw(ctx, userID, order, sum),
		"успешное получение текущей суммы баллов",
	)
	assert.ErrorIs(
		t,
		service.Withdraw(ctx, userID, order, unavailableSum),
		inerr.ErrInsufficientFunds,
		"ошибка при списании: на счету недостаточно средств",
	)

	repository.AssertExpectations(t)
}

func TestTransaction_GetWithdrawals(t *testing.T) {
	var (
		ctx          = context.Background()
		userID       = 1
		errorUserID  = 2
		transactions = []entity.Transaction{
			{
				Order:       "148561163482734",
				Sum:         500,
				ProcessedAt: time.Now(),
			},
			{
				Order:       "267624438264306",
				Sum:         100,
				ProcessedAt: time.Now(),
			},
		}
		repository = &TransactionRepositoryMock{}
	)
	repository.
		On("FindAllByUserID", userID, entity.TransactionTypeOut).
		Return(transactions, nil).
		Once()
	repository.
		On("FindAllByUserID", errorUserID, entity.TransactionTypeOut).
		Return([]entity.Transaction{}, errors.New("")).
		Once()
	service := Transaction{repository: repository}

	resTransactions, _ := service.GetWithdrawals(ctx, userID)
	assert.Equal(t, transactions, resTransactions, "успешное получение списаний")

	_, err := service.GetWithdrawals(ctx, errorUserID)
	assert.Error(t, err, "ошибка при получении списаний")

	repository.AssertExpectations(t)
}
