package handler

import (
	"context"
	"errors"
	inerr "github.com/ivanpodgorny/gophermart/internal/errors"
	"net/http"
)

type Signup struct {
	signuper  Signuper
	validator Validator
}

type Signuper interface {
	Register(ctx context.Context, login, password string) (token string, err error)
	Login(ctx context.Context, login, password string) (token string, err error)
}

func NewSignup(s Signuper, v Validator) *Signup {
	return &Signup{
		signuper:  s,
		validator: v,
	}
}

// Register регистрирует пользователя по паре логин/пароль. В случае успешного
// создания пользователя возвращает ответ с кодом 200 и токен в заголовке Authorization.
func (h *Signup) Register(w http.ResponseWriter, r *http.Request) {
	req := SignupRequest{}
	if err := readJSONBodyAndValidate(r.Context(), &req, r, h.validator); err != nil {
		badRequest(w)

		return
	}

	token, err := h.signuper.Register(r.Context(), req.Login, req.Password)
	status := http.StatusOK
	if errors.Is(err, inerr.ErrUserExists) {
		status = http.StatusConflict
	} else if err != nil {
		serverError(w)

		return
	}

	w.Header().Set("Authorization", token)
	w.WriteHeader(status)
}

// Login аутентифицирует пользователя по паре логин/пароль. В случае успешной
// аутентификации возвращает ответ с кодом 200 и токен в заголовке Authorization.
func (h *Signup) Login(w http.ResponseWriter, r *http.Request) {
	req := SignupRequest{}
	if err := readJSONBodyAndValidate(r.Context(), &req, r, h.validator); err != nil {
		badRequest(w)

		return
	}

	token, err := h.signuper.Login(r.Context(), req.Login, req.Password)
	status := http.StatusOK
	if errors.Is(err, inerr.ErrUserNotFound) {
		status = http.StatusUnauthorized
	} else if err != nil {
		serverError(w)

		return
	}

	w.Header().Set("Authorization", token)
	w.WriteHeader(status)
}
