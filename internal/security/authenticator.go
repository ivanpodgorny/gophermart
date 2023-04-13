package security

import (
	"context"
	"errors"
	"net/http"
)

type Authenticator struct {
	signer  Signer
	storage TokenStorage
}

type TokenStorage interface {
	Save(ctx context.Context, token string, userID int) error
	FindUserID(ctx context.Context, token string) (int, error)
}

type Signer interface {
	Sign(token string) string
	Parse(signed string) (string, error)
}

type userIDContextKey string

const userIDKey userIDContextKey = "currentUserID"

func NewAuthenticator(sgn Signer, store TokenStorage) *Authenticator {
	return &Authenticator{
		signer:  sgn,
		storage: store,
	}
}

// Authenticate проверяет подлинность токена, получает идентификатор пользователя из TokenStorage,
// и устанавливает его в контекст запроса. Если не удается проверить подлинность или найти
// соотвествующую запись в TokenStorage, возвращает ошибку.
func (a *Authenticator) Authenticate(signed string, r *http.Request) (*http.Request, error) {
	token, err := a.signer.Parse(signed)
	if err != nil {
		return r, err
	}

	userID, err := a.storage.FindUserID(r.Context(), token)
	if err != nil {
		return r, err
	}

	return a.setIdentifier(userID, r), nil
}

// GrantToken создает токен для пользователя и сохраняет его в TokenStorage.
// Возвращет токен, подписанный Signer.
func (a *Authenticator) GrantToken(ctx context.Context, userID int) (string, error) {
	token, err := RandomString(32)
	if err != nil {
		return "", err
	}

	if err := a.storage.Save(ctx, token, userID); err != nil {
		return "", err
	}

	return a.signer.Sign(token), nil
}

// UserIdentifier возвращает идентификатор аутентифицированного пользователя из контекста запроса.
func (a *Authenticator) UserIdentifier(r *http.Request) (int, error) {
	val := r.Context().Value(userIDKey)
	if val == nil {
		return 0, errors.New("not found")
	}

	return val.(int), nil
}

func (a *Authenticator) setIdentifier(userID int, r *http.Request) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), userIDKey, userID))
}
