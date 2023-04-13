package handler

import (
	"context"
	"errors"
	"github.com/ivanpodgorny/gophermart/internal/entity"
	inerr "github.com/ivanpodgorny/gophermart/internal/errors"
	"net/http"
)

type Transaction struct {
	processor     TransactionProcessor
	authenticator IdentityProvider
	validator     Validator
}

type TransactionProcessor interface {
	GetBalance(ctx context.Context, userID int) (current float64, withdrawn float64, err error)
	Withdraw(ctx context.Context, userID int, order string, sum float64) error
	GetWithdrawals(ctx context.Context, userID int) ([]entity.Transaction, error)
}

func NewTransaction(p TransactionProcessor, a IdentityProvider, v Validator) *Transaction {
	return &Transaction{
		processor:     p,
		authenticator: a,
		validator:     v,
	}
}

// GetBalance возвращает данные о текущей сумме баллов лояльности пользователя,
// а также сумме  использованных за весь период регистрации баллов, в формате
// {"current": 500.5,  "withdrawn": 42}.
func (h *Transaction) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID, _ := h.authenticator.UserIdentifier(r)
	current, withdrawn, err := h.processor.GetBalance(r.Context(), userID)
	if err != nil {
		serverError(w)

		return
	}

	resp := struct {
		Current   float64 `json:"current"`
		Withdrawn float64 `json:"withdrawn"`
	}{
		Current:   current,
		Withdrawn: withdrawn,
	}
	responseAsJSON(w, resp, http.StatusOK)
}

// Withdraw обрабатывает запрос на списание баллов в счет оплаты заказа.
// Возвращает ответ с кодом 200 в случае успеха. Если на счету недостаточно
// средств, возвращает ответ с кодом 402.
func (h *Transaction) Withdraw(w http.ResponseWriter, r *http.Request) {
	req := WithdrawRequest{}
	if err := readJSONBodyAndValidate(r.Context(), &req, r, h.validator); err != nil {
		badRequest(w)

		return
	}

	if err := h.validator.Var(r.Context(), req.Order, "luhn"); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)

		return
	}

	userID, _ := h.authenticator.UserIdentifier(r)

	err := h.processor.Withdraw(r.Context(), userID, req.Order, req.Sum)
	status := http.StatusOK
	if errors.Is(err, inerr.ErrInsufficientFunds) {
		status = http.StatusPaymentRequired
	} else if err != nil {
		serverError(w)

		return
	}

	w.WriteHeader(status)
}

// GetWithdrawals возвращает списания баллов пользвателя. Если списаний нет,
// возвращает ответ с кодом 204.
func (h *Transaction) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID, _ := h.authenticator.UserIdentifier(r)

	transactions, err := h.processor.GetWithdrawals(r.Context(), userID)
	if err != nil {
		serverError(w)

		return
	}

	if len(transactions) == 0 {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	responseAsJSON(w, transactions, http.StatusOK)
}
