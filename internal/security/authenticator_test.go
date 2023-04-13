package security

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http/httptest"
	"testing"
)

type SignerMock struct {
	mock.Mock
}

func (m *SignerMock) Sign(_ string) string {
	args := m.Called()

	return args.String(0)
}

func (m *SignerMock) Parse(signed string) (string, error) {
	args := m.Called(signed)

	return args.String(0), args.Error(1)
}

type TokenStorageMock struct {
	mock.Mock
}

func (m *TokenStorageMock) Save(_ context.Context, _ string, userID int) error {
	args := m.Called(userID)

	return args.Error(0)
}

func (m *TokenStorageMock) FindUserID(_ context.Context, token string) (int, error) {
	args := m.Called(token)

	return args.Int(0), args.Error(1)
}

func TestAuthenticator_Authenticate(t *testing.T) {
	var (
		signed            = "signed"
		nonexistentSigned = "nonexistentSigned"
		invalidSigned     = "invalidSigned"
		token             = "token"
		nonexistentToken  = "nonexistentToken"
		userID            = 1
		request           = httptest.NewRequest("", "/", nil)
		signer            = &SignerMock{}
		storage           = &TokenStorageMock{}
	)
	signer.On("Parse", signed).Return(token, nil).Once()
	signer.On("Parse", invalidSigned).Return("", errors.New("")).Once()
	signer.On("Parse", nonexistentSigned).Return(nonexistentToken, nil).Once()
	storage.On("FindUserID", token).Return(userID, nil).Once()
	storage.On("FindUserID", nonexistentToken).Return(0, errors.New("")).Once()
	authenticator := NewAuthenticator(signer, storage)

	_, err := authenticator.UserIdentifier(request)
	assert.Error(t, err, "неаутентифицированный пользователь")

	_, err = authenticator.Authenticate(invalidSigned, request)
	assert.Error(t, err, "невалидный токен")

	_, err = authenticator.Authenticate(nonexistentSigned, request)
	assert.Error(t, err, "несуществующий токен")

	request, _ = authenticator.Authenticate(signed, request)
	id, _ := authenticator.UserIdentifier(request)
	assert.Equal(t, userID, id, "успешная аутентификация")

	signer.AssertExpectations(t)
	storage.AssertExpectations(t)
}

func TestAuthenticator_GrantToken(t *testing.T) {
	var (
		token     = "token"
		userID    = 1
		errUserID = 2
		ctx       = context.Background()
		signer    = &SignerMock{}
		storage   = &TokenStorageMock{}
	)
	signer.On("Sign").Return(token).Once()
	storage.On("Save", userID).Return(nil).Once()
	storage.On("Save", errUserID).Return(errors.New("")).Once()
	authenticator := NewAuthenticator(signer, storage)

	signed, _ := authenticator.GrantToken(ctx, userID)
	assert.Equal(t, token, signed, "успешное создание токена")

	_, err := authenticator.GrantToken(ctx, errUserID)
	assert.Error(t, err, "ошибка при сохранении токена")

	signer.AssertExpectations(t)
	storage.AssertExpectations(t)
}
