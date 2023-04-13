package validator

import (
	"context"
	"errors"
	v10validator "github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

type EngineMock struct {
	mock.Mock
}

func (m *EngineMock) StructCtx(_ context.Context, s any) error {
	args := m.Called(s)

	return args.Error(0)
}

func (m *EngineMock) VarCtx(_ context.Context, field any, tag string) error {
	args := m.Called(field, tag)

	return args.Error(0)
}

func TestValidator_Struct(t *testing.T) {
	type ValidatedStruct struct {
		Name string `validate:"required"`
	}

	var (
		ctx           = context.Background()
		engine        = &EngineMock{}
		validStruct   = &ValidatedStruct{Name: "name"}
		invalidStruct = &ValidatedStruct{}
	)
	engine.On("StructCtx", validStruct).Return(nil).Once()
	engine.On("StructCtx", invalidStruct).Return(errors.New("")).Once()
	v := &Validator{engine: engine}

	assert.NoError(t, v.Struct(ctx, validStruct))
	assert.Error(t, v.Struct(ctx, invalidStruct))
	engine.AssertExpectations(t)
}

func TestValidator_Var(t *testing.T) {
	var (
		ctx        = context.Background()
		engine     = &EngineMock{}
		tag        = "alnum"
		validStr   = "name"
		invalidStr = "name$%/"
	)
	engine.On("VarCtx", validStr, tag).Return(nil).Once()
	engine.On("VarCtx", invalidStr, tag).Return(errors.New("")).Once()
	v := &Validator{engine: engine}

	assert.NoError(t, v.Var(ctx, validStr, tag))
	assert.Error(t, v.Var(ctx, invalidStr, tag))
	engine.AssertExpectations(t)
}

func TestLuhn(t *testing.T) {
	var (
		ctx = context.Background()
		v10 = v10validator.New()
		tag = "luhn"
	)
	require.NoError(t, v10.RegisterValidation(tag, Luhn))
	v := New(v10)

	tests := []struct {
		name   string
		number string
		valid  bool
	}{
		{
			name:   "верная контрольная цифра",
			number: "166221614883769",
			valid:  true,
		},
		{
			name:   "неверная контрольная цифра",
			number: "166221614883768",
			valid:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, v.Var(ctx, tt.number, tag) == nil)
		})
	}
}
