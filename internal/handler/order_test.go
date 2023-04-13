package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

type OrderProcessorMock struct {
	mock.Mock
}

func (m *OrderProcessorMock) Create(_ context.Context, userID int, num string) error {
	args := m.Called(userID, num)

	return args.Error(0)
}

func (m *OrderProcessorMock) GetAll(_ context.Context, userID int) ([]entity.Order, error) {
	args := m.Called(userID)

	return args.Get(0).([]entity.Order), args.Error(1)
}

func TestOrder_CreateSuccess(t *testing.T) {
	var (
		num           = "166221614883769"
		userID        = 1
		processor     = &OrderProcessorMock{}
		authenticator = &AuthenticatorMock{}
		val           = &ValidatorMock{}
	)

	val.On("Var", num, "luhn").Return(nil).Once()
	authenticator.On("UserIdentifier").Return(userID, nil).Once()
	processor.On("Create", userID, num).Return(nil).Once()
	handler := Order{
		processor:     processor,
		authenticator: authenticator,
		validator:     val,
	}

	result := sendTestRequest(
		http.MethodPost,
		bytes.NewBuffer([]byte(num)),
		handler.Create,
	)
	assert.Equal(t, http.StatusAccepted, result.StatusCode)
	require.NoError(t, result.Body.Close())
	val.AssertExpectations(t)
	processor.AssertExpectations(t)
	authenticator.AssertExpectations(t)
}

func TestOrder_CreateProcessorErrors(t *testing.T) {
	var (
		num                = "166221614883769"
		userID             = 1
		processorExists    = &OrderProcessorMock{}
		processorNotBelong = &OrderProcessorMock{}
		processorError     = &OrderProcessorMock{}
		authenticator      = &AuthenticatorMock{}
		val                = &ValidatorMock{}
	)

	val.On("Var", num, "luhn").Return(nil).Times(3)
	authenticator.On("UserIdentifier").Return(userID, nil).Times(3)
	processorExists.
		On("Create", userID, num).
		Return(inerr.ErrOrderExists).
		Once()
	processorNotBelong.
		On("Create", userID, num).
		Return(inerr.ErrOrderNotBelongToUser).
		Once()
	processorError.
		On("Create", userID, num).
		Return(errors.New("")).
		Once()

	tests := []struct {
		name           string
		processor      OrderProcessor
		wantStatusCode int
	}{
		{
			name:           "номер заказа уже был загружен пользователем",
			processor:      processorExists,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "номер заказа уже был загружен другим пользователем",
			processor:      processorNotBelong,
			wantStatusCode: http.StatusConflict,
		},
		{
			name:           "ошибка при создании заказа",
			processor:      processorError,
			wantStatusCode: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Order{
				processor:     tt.processor,
				authenticator: authenticator,
				validator:     val,
			}
			result := sendTestRequest(
				http.MethodPost,
				bytes.NewBuffer([]byte(num)),
				handler.Create,
			)
			assert.Equal(t, tt.wantStatusCode, result.StatusCode)
			require.NoError(t, result.Body.Close())
		})
	}
	val.AssertExpectations(t)
	processorExists.AssertExpectations(t)
	processorNotBelong.AssertExpectations(t)
	processorError.AssertExpectations(t)
	authenticator.AssertExpectations(t)
}

func TestOrder_CreateValidationErrors(t *testing.T) {
	var (
		processor     = &OrderProcessorMock{}
		authenticator = &AuthenticatorMock{}
		v10           = v10validator.New()
	)
	require.NoError(t, v10.RegisterValidation("luhn", validator.Luhn))
	handler := Order{
		processor:     processor,
		authenticator: authenticator,
		validator:     validator.New(v10),
	}
	result := sendTestRequest(
		http.MethodPost,
		bytes.NewBuffer([]byte("166221614883768")),
		handler.Create,
	)
	assert.Equal(t, http.StatusUnprocessableEntity, result.StatusCode)
	require.NoError(t, result.Body.Close())
	processor.AssertExpectations(t)
	authenticator.AssertExpectations(t)
}

func TestOrder_GetAllSuccess(t *testing.T) {
	var (
		userID        = 1
		processor     = &OrderProcessorMock{}
		authenticator = &AuthenticatorMock{}
		orders        = []entity.Order{
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
	)

	authenticator.On("UserIdentifier").Return(userID, nil).Once()
	processor.On("GetAll", userID).Return(orders, nil).Once()
	handler := Order{
		processor:     processor,
		authenticator: authenticator,
	}
	result := sendTestRequest(
		http.MethodGet,
		nil,
		handler.GetAll,
	)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	b, err := io.ReadAll(result.Body)
	require.NoError(t, err)
	ordersJSON, err := json.Marshal(orders)
	require.NoError(t, err)
	assert.JSONEq(t, string(ordersJSON), string(b))
	require.NoError(t, result.Body.Close())
	authenticator.AssertExpectations(t)
	processor.AssertExpectations(t)
}

func TestOrder_GetAllProcessorErrors(t *testing.T) {
	var (
		userID             = 1
		processorError     = &OrderProcessorMock{}
		processorNoContent = &OrderProcessorMock{}
		authenticator      = &AuthenticatorMock{}
	)

	authenticator.On("UserIdentifier").Return(userID, nil).Twice()
	processorError.
		On("GetAll", userID).
		Return([]entity.Order{}, errors.New("")).
		Once()
	processorNoContent.
		On("GetAll", userID).
		Return([]entity.Order{}, nil).
		Once()

	tests := []struct {
		name           string
		processor      OrderProcessor
		wantStatusCode int
	}{
		{
			name:           "ошибка при получении списка заказов пользователя",
			processor:      processorError,
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name:           "пустой список заказов пользователя",
			processor:      processorNoContent,
			wantStatusCode: http.StatusNoContent,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Order{
				processor:     tt.processor,
				authenticator: authenticator,
			}
			result := sendTestRequest(
				http.MethodGet,
				nil,
				handler.GetAll,
			)
			assert.Equal(t, tt.wantStatusCode, result.StatusCode)
			require.NoError(t, result.Body.Close())
		})
	}
	authenticator.AssertExpectations(t)
	processorError.AssertExpectations(t)
}
