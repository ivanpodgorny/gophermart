package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
)

type SignupRequest struct {
	Login    string `json:"login" validate:"required,alphanum,min=3,max=20"`
	Password string `json:"password" validate:"required,min=8,max=32"`
}

type WithdrawRequest struct {
	Order string  `json:"order" validate:"required"`
	Sum   float64 `json:"sum" validate:"required,min=1"`
}

type IdentityProvider interface {
	UserIdentifier(*http.Request) (int, error)
}

type Validator interface {
	Struct(ctx context.Context, s any) error
	Var(ctx context.Context, field any, tag string) error
}

func readJSONBody(v any, r *http.Request) error {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, v)
}

func readJSONBodyAndValidate(ctx context.Context, v any, r *http.Request, validator Validator) error {
	if err := readJSONBody(v, r); err != nil {
		return err
	}

	return validator.Struct(ctx, v)
}
