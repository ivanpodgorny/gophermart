package handler

import (
	"context"
	"errors"
	"github.com/ivanpodgorny/gophermart/internal/entity"
	inerr "github.com/ivanpodgorny/gophermart/internal/errors"
	"io"
	"net/http"
)

type Order struct {
	processor     OrderProcessor
	authenticator IdentityProvider
	validator     Validator
}

type OrderProcessor interface {
	Create(ctx context.Context, userID int, num string) error
	GetAll(ctx context.Context, userID int) ([]entity.Order, error)
}

func NewOrder(p OrderProcessor, a IdentityProvider, v Validator) *Order {
	return &Order{
		processor:     p,
		authenticator: a,
		validator:     v,
	}
}

// Create обрабатывает запрос на добавление нового заказа. Возвращает ответ с кодом 202,
// если заказ принят в обработку, 200 - если заказ уже был загружен пользователем.
func (h *Order) Create(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		badRequest(w)

		return
	}

	num := string(b)
	if err := h.validator.Var(r.Context(), num, "luhn"); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)

		return
	}

	userID, _ := h.authenticator.UserIdentifier(r)

	err = h.processor.Create(r.Context(), userID, num)
	status := http.StatusAccepted
	if errors.Is(err, inerr.ErrOrderExists) {
		status = http.StatusOK
	} else if errors.Is(err, inerr.ErrOrderNotBelongToUser) {
		status = http.StatusConflict
	} else if err != nil {
		serverError(w)

		return
	}

	w.WriteHeader(status)
}

// GetAll возвращает список загруженных заказов пользователя. Если заказов нет,
// возвращает ответ с кодом 204.
func (h *Order) GetAll(w http.ResponseWriter, r *http.Request) {
	userID, _ := h.authenticator.UserIdentifier(r)

	orders, err := h.processor.GetAll(r.Context(), userID)
	if err != nil {
		serverError(w)

		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	responseAsJSON(w, orders, http.StatusOK)
}
