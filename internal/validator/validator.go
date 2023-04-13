package validator

import (
	"context"
	v10validator "github.com/go-playground/validator/v10"
	"reflect"
	"strconv"
)

type Validator struct {
	engine Engine
}

type Engine interface {
	StructCtx(ctx context.Context, s any) error
	VarCtx(ctx context.Context, field any, tag string) error
}

func New(e Engine) *Validator {
	return &Validator{engine: e}
}

func (v *Validator) Struct(ctx context.Context, s any) error {
	return v.engine.StructCtx(ctx, s)
}

func (v *Validator) Var(ctx context.Context, field any, tag string) error {
	return v.engine.VarCtx(ctx, field, tag)
}

func Luhn(fl v10validator.FieldLevel) bool {
	var (
		val        = fl.Field()
		luhn int64 = 0
	)
	if val.Kind() != reflect.String {
		return false
	}

	nStr := val.String()
	cd, err := strconv.ParseInt(nStr[len(nStr)-1:], 10, 64)
	if err != nil {
		return false
	}

	number, err := strconv.ParseInt(nStr[:len(nStr)-1], 10, 64)
	if err != nil {
		return false
	}

	for i := 0; number > 0; i++ {
		cur := number % 10

		if i%2 == 0 {
			cur = cur * 2
			if cur > 9 {
				cur = cur%10 + cur/10
			}
		}

		luhn += cur
		number = number / 10
	}

	ccd := luhn % 10
	if ccd != 0 {
		ccd = 10 - ccd
	}

	return ccd == cd
}
