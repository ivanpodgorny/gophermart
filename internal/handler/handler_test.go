package handler

import (
	"context"
	"github.com/stretchr/testify/mock"
	"io"
	"net/http"
	"net/http/httptest"
)

type ValidatorMock struct {
	mock.Mock
}

func (m *ValidatorMock) Struct(_ context.Context, s any) error {
	args := m.Called(s)

	return args.Error(0)
}

func (m *ValidatorMock) Var(_ context.Context, field any, tag string) error {
	args := m.Called(field, tag)

	return args.Error(0)
}

type AuthenticatorMock struct {
	mock.Mock
}

func (m *AuthenticatorMock) UserIdentifier(_ *http.Request) (int, error) {
	args := m.Called()

	return args.Int(0), args.Error(1)
}

func sendTestRequest(method string, body io.Reader, handler http.HandlerFunc) *http.Response {
	request := httptest.NewRequest(method, "/", body)
	w := httptest.NewRecorder()
	handler(w, request)

	return w.Result()
}
