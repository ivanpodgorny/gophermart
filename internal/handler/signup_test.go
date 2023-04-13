package handler

import (
	"bytes"
	"context"
	"errors"
	v10validator "github.com/go-playground/validator/v10"
	inerr "github.com/ivanpodgorny/gophermart/internal/errors"
	"github.com/ivanpodgorny/gophermart/internal/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

type SignuperMock struct {
	mock.Mock
}

func (m *SignuperMock) Register(_ context.Context, login, password string) (string, error) {
	args := m.Called(login, password)

	return args.String(0), args.Error(1)
}

func (m *SignuperMock) Login(_ context.Context, login, password string) (string, error) {
	args := m.Called(login, password)

	return args.String(0), args.Error(1)
}

func TestSignUp_RegisterSuccess(t *testing.T) {
	var (
		login    = "login"
		password = "password"
		token    = "token"
		signuper = &SignuperMock{}
		val      = &ValidatorMock{}
	)

	val.On("Struct", &SignupRequest{Login: login, Password: password}).Return(nil).Once()
	signuper.On("Register", login, password).Return(token, nil).Once()
	handler := Signup{
		signuper:  signuper,
		validator: val,
	}

	result := sendTestRequest(
		http.MethodPost,
		bytes.NewBuffer([]byte(`{"login": "`+login+`","password": "`+password+`"}`)),
		handler.Register,
	)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Equal(t, token, result.Header.Get("Authorization"))
	require.NoError(t, result.Body.Close())
	val.AssertExpectations(t)
	signuper.AssertExpectations(t)
}

func TestSignUp_RegisterSignuperErrors(t *testing.T) {
	var (
		login            = "login"
		password         = "password"
		signuperConflict = &SignuperMock{}
		signuperError    = &SignuperMock{}
		val              = &ValidatorMock{}
	)

	val.On("Struct", &SignupRequest{Login: login, Password: password}).Return(nil).Twice()
	signuperConflict.
		On("Register", login, password).
		Return("", inerr.ErrUserExists).
		Once()
	signuperError.
		On("Register", login, password).
		Return("", errors.New("")).
		Once()

	tests := []struct {
		name           string
		signuper       Signuper
		wantStatusCode int
	}{
		{
			name:           "логин уже занят",
			signuper:       signuperConflict,
			wantStatusCode: http.StatusConflict,
		},
		{
			name:           "ошибка при создании пользователя",
			signuper:       signuperError,
			wantStatusCode: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Signup{
				signuper:  tt.signuper,
				validator: val,
			}
			result := sendTestRequest(
				http.MethodPost,
				bytes.NewBuffer([]byte(`{"login": "`+login+`","password": "`+password+`"}`)),
				handler.Register,
			)
			assert.Equal(t, tt.wantStatusCode, result.StatusCode)
			require.NoError(t, result.Body.Close())
		})
	}
	val.AssertExpectations(t)
	signuperConflict.AssertExpectations(t)
	signuperError.AssertExpectations(t)
}

func TestSignUp_RegisterValidationErrors(t *testing.T) {
	signuper := &SignuperMock{}
	handler := Signup{
		signuper:  signuper,
		validator: validator.New(v10validator.New()),
	}

	tests := []struct {
		name string
		body string
	}{
		{
			name: "не передан логин",
			body: `{"password": "password"}`,
		},
		{
			name: "не передан пароль",
			body: `{"login": "login"}`,
		},
		{
			name: "слишком длинный логин",
			body: `{"login": "loginloginloginlogin1", "password": "password"}`,
		},
		{
			name: "слишком длинный пароль",
			body: `{"login": "login", "password": "passwordpasswordpasswordpassword1"}`,
		},
		{
			name: "слишком короткий логин",
			body: `{"login": "lo", "password": "password"}`,
		},
		{
			name: "слишком короткий пароль",
			body: `{"login": "login", "password": "passwor"}`,
		},
		{
			name: "недопустимые символы в логине",
			body: `{"login": "lo%$/", "password": "password"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sendTestRequest(
				http.MethodPost,
				bytes.NewBuffer([]byte(tt.body)),
				handler.Register,
			)
			assert.Equal(t, http.StatusBadRequest, result.StatusCode)
			require.NoError(t, result.Body.Close())
		})
	}
	signuper.AssertExpectations(t)
}

func TestSignUp_LoginSuccess(t *testing.T) {
	var (
		login    = "login"
		password = "password"
		token    = "token"
		signuper = &SignuperMock{}
		val      = &ValidatorMock{}
	)

	val.On("Struct", &SignupRequest{Login: login, Password: password}).Return(nil).Once()
	signuper.On("Login", login, password).Return(token, nil).Once()
	handler := Signup{
		signuper:  signuper,
		validator: val,
	}

	result := sendTestRequest(
		http.MethodPost,
		bytes.NewBuffer([]byte(`{"login": "`+login+`","password": "`+password+`"}`)),
		handler.Login,
	)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Equal(t, token, result.Header.Get("Authorization"))
	require.NoError(t, result.Body.Close())
	val.AssertExpectations(t)
	signuper.AssertExpectations(t)
}

func TestSignUp_LoginSignuperErrors(t *testing.T) {
	var (
		login            = "login"
		password         = "password"
		signuperNotFound = &SignuperMock{}
		signuperError    = &SignuperMock{}
		val              = &ValidatorMock{}
	)

	val.On("Struct", &SignupRequest{Login: login, Password: password}).Return(nil).Twice()
	signuperNotFound.
		On("Login", login, password).
		Return("", inerr.ErrUserNotFound).
		Once()
	signuperError.
		On("Login", login, password).
		Return("", errors.New("")).
		Once()

	tests := []struct {
		name           string
		signuper       Signuper
		wantStatusCode int
	}{
		{
			name:           "неверный логин или пароль",
			signuper:       signuperNotFound,
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "ошибка при логине пользователя",
			signuper:       signuperError,
			wantStatusCode: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Signup{
				signuper:  tt.signuper,
				validator: val,
			}
			result := sendTestRequest(
				http.MethodPost,
				bytes.NewBuffer([]byte(`{"login": "`+login+`","password": "`+password+`"}`)),
				handler.Login,
			)
			assert.Equal(t, tt.wantStatusCode, result.StatusCode)
			require.NoError(t, result.Body.Close())
		})
	}
	val.AssertExpectations(t)
	signuperNotFound.AssertExpectations(t)
	signuperError.AssertExpectations(t)
}

func TestSignUp_LoginValidationErrors(t *testing.T) {
	signuper := &SignuperMock{}
	handler := Signup{
		signuper:  signuper,
		validator: validator.New(v10validator.New()),
	}

	tests := []struct {
		name string
		body string
	}{
		{
			name: "не передан логин",
			body: `{"password": "password"}`,
		},
		{
			name: "не передан пароль",
			body: `{"login": "login"}`,
		},
		{
			name: "слишком длинный логин",
			body: `{"login": "loginloginloginlogin1", "password": "password"}`,
		},
		{
			name: "слишком длинный пароль",
			body: `{"login": "login", "password": "passwordpasswordpasswordpassword1"}`,
		},
		{
			name: "слишком короткий логин",
			body: `{"login": "lo", "password": "password"}`,
		},
		{
			name: "слишком короткий пароль",
			body: `{"login": "login", "password": "passwor"}`,
		},
		{
			name: "недопустимые символы в логине",
			body: `{"login": "lo%$/", "password": "password"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sendTestRequest(
				http.MethodPost,
				bytes.NewBuffer([]byte(tt.body)),
				handler.Login,
			)
			assert.Equal(t, http.StatusBadRequest, result.StatusCode)
			require.NoError(t, result.Body.Close())
		})
	}
	signuper.AssertExpectations(t)
}
