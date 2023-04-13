package service

import (
	"context"
	"errors"
	inerr "github.com/ivanpodgorny/gophermart/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

type UserRepositoryMock struct {
	mock.Mock
}

func (m *UserRepositoryMock) Create(_ context.Context, login, passwordHash string) (int, error) {
	args := m.Called(login, passwordHash)

	return args.Int(0), args.Error(1)
}

func (m *UserRepositoryMock) FindByLogin(_ context.Context, login string) (int, string, error) {
	args := m.Called(login)

	return args.Int(0), args.String(1), args.Error(2)
}

type HasherMock struct {
	mock.Mock
}

func (m *HasherMock) Hash(password string) (string, error) {
	args := m.Called(password)

	return args.String(0), args.Error(1)
}

func (m *HasherMock) Compare(password, hash string) bool {
	args := m.Called(password, hash)

	return args.Bool(0)
}

type TokenProviderMock struct {
	mock.Mock
}

func (m *TokenProviderMock) GrantToken(_ context.Context, userID int) (string, error) {
	args := m.Called(userID)

	return args.String(0), args.Error(1)
}

func TestSignup_Register(t *testing.T) {
	var (
		ctx             = context.Background()
		userID          = 1
		errorUserID     = 2
		login           = "login"
		duplicatedLogin = "duplicatedLogin"
		errorUserLogin  = "errorUserLogin"
		password        = "password"
		errorPassword   = "errorPassword"
		passwordHash    = "passwordHash"
		token           = "token"
		repository      = &UserRepositoryMock{}
		hasher          = &HasherMock{}
		tokenProvider   = &TokenProviderMock{}
	)
	hasher.On("Hash", password).Return(passwordHash, nil).Times(3)
	hasher.On("Hash", errorPassword).Return("", errors.New("")).Once()
	repository.On("Create", login, passwordHash).Return(userID, nil).Once()
	repository.On("Create", duplicatedLogin, passwordHash).Return(0, inerr.ErrUserExists).Once()
	repository.On("Create", errorUserLogin, passwordHash).Return(errorUserID, nil).Once()
	tokenProvider.On("GrantToken", userID).Return(token, nil).Once()
	tokenProvider.On("GrantToken", errorUserID).Return("", errors.New("")).Once()
	service := Signup{
		repository:    repository,
		hasher:        hasher,
		tokenProvider: tokenProvider,
	}

	grantedToken, _ := service.Register(ctx, login, password)
	assert.Equal(t, token, grantedToken, "успешная регистрация")

	_, err := service.Register(ctx, login, errorPassword)
	assert.Error(t, err, "ошибка при создании хэша пароля")

	_, err = service.Register(ctx, duplicatedLogin, password)
	assert.ErrorIs(t, err, inerr.ErrUserExists, "регистрация с существующим логином")

	_, err = service.Register(ctx, errorUserLogin, password)
	assert.Error(t, err, "ошибка при создании токена")

	repository.AssertExpectations(t)
	hasher.AssertExpectations(t)
	tokenProvider.AssertExpectations(t)
}

func TestSignup_Login(t *testing.T) {
	var (
		ctx            = context.Background()
		userID         = 1
		errorUserID    = 2
		login          = "login"
		wrongLogin     = "wrongLogin"
		errorUserLogin = "errorUserLogin"
		password       = "password"
		wrongPassword  = "wrongPassword"
		passwordHash   = "passwordHash"
		token          = "token"
		repository     = &UserRepositoryMock{}
		hasher         = &HasherMock{}
		tokenProvider  = &TokenProviderMock{}
	)
	repository.On("FindByLogin", login).Return(userID, passwordHash, nil).Twice()
	repository.On("FindByLogin", errorUserLogin).Return(errorUserID, passwordHash, nil).Once()
	repository.On("FindByLogin", wrongLogin).Return(0, "", errors.New("")).Once()
	hasher.On("Compare", password, passwordHash).Return(true).Twice()
	hasher.On("Compare", wrongPassword, passwordHash).Return(false).Once()
	tokenProvider.On("GrantToken", userID).Return(token, nil).Once()
	tokenProvider.On("GrantToken", errorUserID).Return("", errors.New("")).Once()
	service := Signup{
		repository:    repository,
		hasher:        hasher,
		tokenProvider: tokenProvider,
	}

	grantedToken, _ := service.Login(ctx, login, password)
	assert.Equal(t, token, grantedToken, "успешная аутентификация")

	_, err := service.Login(ctx, wrongLogin, password)
	assert.ErrorIs(t, err, inerr.ErrUserNotFound, "неверный логин")

	_, err = service.Login(ctx, login, wrongPassword)
	assert.ErrorIs(t, err, inerr.ErrUserNotFound, "неверный пароль")

	_, err = service.Login(ctx, errorUserLogin, password)
	assert.Error(t, err, "ошибка при создании токена")

	repository.AssertExpectations(t)
	hasher.AssertExpectations(t)
	tokenProvider.AssertExpectations(t)
}
