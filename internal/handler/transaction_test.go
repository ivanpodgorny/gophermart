package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	v10validator "github.com/go-playground/validator/v10"
	"github.com/ivanpodgorny/gophermart/internal/entity"
	inerr "github.com/ivanpodgorny/gophermart/internal/errors"
	"github.com/ivanpodgorny/gophermart/internal/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
	"time"
)

type TransactionProcessorMock struct {
	mock.Mock
}

func (m *TransactionProcessorMock) GetBalance(_ context.Context, userID int) (float64, float64, error) {
	args := m.Called(userID)

	return args.Get(0).(float64), args.Get(1).(float64), args.Error(2)
}

func (m *TransactionProcessorMock) Withdraw(_ context.Context, userID int, order string, sum float64) error {
	args := m.Called(userID, order, sum)

	return args.Error(0)
}

func (m *TransactionProcessorMock) GetWithdrawals(_ context.Context, userID int) ([]entity.Transaction, error) {
	args := m.Called(userID)

	return args.Get(0).([]entity.Transaction), args.Error(1)
}

func TestTransaction_GetBalanceSuccess(t *testing.T) {
	var (
		userID        = 1
		current       = 500.5
		withdrawn     = 42.0
		processor     = &TransactionProcessorMock{}
		authenticator = &AuthenticatorMock{}
	)

	authenticator.On("UserIdentifier").Return(userID, nil).Once()
	processor.On("GetBalance", userID).Return(current, withdrawn, nil).Once()
	handler := Transaction{
		processor:     processor,
		authenticator: authenticator,
	}

	result := sendTestRequest(
		http.MethodGet,
		nil,
		handler.GetBalance,
	)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	b, err := io.ReadAll(result.Body)
	require.NoError(t, err)
	assert.JSONEq(t, fmt.Sprintf(`{"current": %f, "withdrawn": %f}`, current, withdrawn), string(b))
	require.NoError(t, result.Body.Close())
	processor.AssertExpectations(t)
	authenticator.AssertExpectations(t)
}

func TestTransaction_GetBalanceProcessorErrors(t *testing.T) {
	var (
		userID        = 1
		processor     = &TransactionProcessorMock{}
		authenticator = &AuthenticatorMock{}
	)

	authenticator.On("UserIdentifier").Return(userID, nil).Once()
	processor.
		On("GetBalance", userID).
		Return(0.0, 0.0, errors.New("")).
		Once()

	handler := Transaction{
		processor:     processor,
		authenticator: authenticator,
	}
	result := sendTestRequest(
		http.MethodGet,
		nil,
		handler.GetBalance,
	)
	assert.Equal(t, http.StatusInternalServerError, result.StatusCode)
	require.NoError(t, result.Body.Close())
	processor.AssertExpectations(t)
	authenticator.AssertExpectations(t)
}

func TestTransaction_WithdrawSuccess(t *testing.T) {
	var (
		userID        = 1
		order         = "166221614883769"
		sum           = 100.0
		processor     = &TransactionProcessorMock{}
		authenticator = &AuthenticatorMock{}
		val           = &ValidatorMock{}
	)

	val.On("Struct", &WithdrawRequest{Order: order, Sum: sum}).Return(nil).Once()
	val.On("Var", order, "luhn").Return(nil).Once()
	authenticator.On("UserIdentifier").Return(userID, nil).Once()
	processor.On("Withdraw", userID, order, sum).Return(nil).Once()
	handler := Transaction{
		processor:     processor,
		authenticator: authenticator,
		validator:     val,
	}

	result := sendTestRequest(
		http.MethodPost,
		bytes.NewBuffer([]byte(fmt.Sprintf(`{"order": "%s", "sum": %f}`, order, sum))),
		handler.Withdraw,
	)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	require.NoError(t, result.Body.Close())
	val.AssertExpectations(t)
	processor.AssertExpectations(t)
	authenticator.AssertExpectations(t)
}

func TestTransaction_WithdrawProcessorErrors(t *testing.T) {
	var (
		userID                = 1
		order                 = "166221614883769"
		sum                   = 100.0
		processorInsufficient = &TransactionProcessorMock{}
		processorError        = &TransactionProcessorMock{}
		authenticator         = &AuthenticatorMock{}
		val                   = &ValidatorMock{}
	)

	val.On("Struct", &WithdrawRequest{Order: order, Sum: sum}).Return(nil).Twice()
	val.On("Var", order, "luhn").Return(nil).Twice()
	authenticator.On("UserIdentifier").Return(userID, nil).Twice()
	processorInsufficient.
		On("Withdraw", userID, order, sum).
		Return(inerr.ErrInsufficientFunds).
		Once()
	processorError.
		On("Withdraw", userID, order, sum).
		Return(errors.New("")).
		Once()

	tests := []struct {
		name           string
		processor      TransactionProcessor
		wantStatusCode int
	}{
		{
			name:           "на счету недостаточно средств",
			processor:      processorInsufficient,
			wantStatusCode: http.StatusPaymentRequired,
		},
		{
			name:           "ошибка при списании баллов",
			processor:      processorError,
			wantStatusCode: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Transaction{
				processor:     tt.processor,
				authenticator: authenticator,
				validator:     val,
			}
			result := sendTestRequest(
				http.MethodPost,
				bytes.NewBuffer([]byte(fmt.Sprintf(`{"order": "%s", "sum": %f}`, order, sum))),
				handler.Withdraw,
			)
			assert.Equal(t, tt.wantStatusCode, result.StatusCode)
			require.NoError(t, result.Body.Close())
		})
	}
	val.AssertExpectations(t)
	processorInsufficient.AssertExpectations(t)
	processorError.AssertExpectations(t)
	authenticator.AssertExpectations(t)
}

func TestTransaction_WithdrawValidationErrors(t *testing.T) {
	var (
		processor     = &TransactionProcessorMock{}
		authenticator = &AuthenticatorMock{}
		v10           = v10validator.New()
	)
	require.NoError(t, v10.RegisterValidation("luhn", validator.Luhn))
	handler := Transaction{
		processor:     processor,
		authenticator: authenticator,
		validator:     validator.New(v10),
	}

	tests := []struct {
		name           string
		body           string
		wantStatusCode int
	}{
		{
			name:           "не передан номер заказа",
			body:           `{"sum": 100}`,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "не передана сумма",
			body:           `{"order": "166221614883769"}`,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "некорректная сумма",
			body:           `{"order": "166221614883769", "sum": -1}`,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "неверный номер заказа",
			body:           `{"order": "166221614883768", "sum": 100}`,
			wantStatusCode: http.StatusUnprocessableEntity,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sendTestRequest(
				http.MethodPost,
				bytes.NewBuffer([]byte(tt.body)),
				handler.Withdraw,
			)
			assert.Equal(t, tt.wantStatusCode, result.StatusCode)
			require.NoError(t, result.Body.Close())
		})
	}
	processor.AssertExpectations(t)
	authenticator.AssertExpectations(t)
}

func TestTransaction_GetWithdrawalsSuccess(t *testing.T) {
	var (
		userID        = 1
		processor     = &TransactionProcessorMock{}
		authenticator = &AuthenticatorMock{}
		transactions  = []entity.Transaction{
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
	)

	authenticator.On("UserIdentifier").Return(userID, nil).Once()
	processor.On("GetWithdrawals", userID).Return(transactions, nil).Once()
	handler := Transaction{
		processor:     processor,
		authenticator: authenticator,
	}
	result := sendTestRequest(
		http.MethodGet,
		nil,
		handler.GetWithdrawals,
	)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	b, err := io.ReadAll(result.Body)
	require.NoError(t, err)
	transactionsJSON, err := json.Marshal(transactions)
	require.NoError(t, err)
	assert.JSONEq(t, string(transactionsJSON), string(b))
	require.NoError(t, result.Body.Close())
	authenticator.AssertExpectations(t)
	processor.AssertExpectations(t)
}

func TestTransaction_GetWithdrawalsProcessorErrors(t *testing.T) {
	var (
		userID             = 1
		processorError     = &TransactionProcessorMock{}
		processorNoContent = &TransactionProcessorMock{}
		authenticator      = &AuthenticatorMock{}
	)

	authenticator.On("UserIdentifier").Return(userID, nil).Twice()
	processorError.
		On("GetWithdrawals", userID).
		Return([]entity.Transaction{}, errors.New("")).
		Once()
	processorNoContent.
		On("GetWithdrawals", userID).
		Return([]entity.Transaction{}, nil).
		Once()

	tests := []struct {
		name           string
		processor      TransactionProcessor
		wantStatusCode int
	}{
		{
			name:           "ошибка при получении списаний пользователя",
			processor:      processorError,
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name:           "пустой список списаний пользователя",
			processor:      processorNoContent,
			wantStatusCode: http.StatusNoContent,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Transaction{
				processor:     tt.processor,
				authenticator: authenticator,
			}
			result := sendTestRequest(
				http.MethodGet,
				nil,
				handler.GetWithdrawals,
			)
			assert.Equal(t, tt.wantStatusCode, result.StatusCode)
			require.NoError(t, result.Body.Close())
		})
	}
	authenticator.AssertExpectations(t)
	processorError.AssertExpectations(t)
}
