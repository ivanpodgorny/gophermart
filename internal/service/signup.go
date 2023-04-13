package service

import (
	"context"
	inerr "github.com/ivanpodgorny/gophermart/internal/errors"
)

type Signup struct {
	repository    UserRepository
	hasher        Hasher
	tokenProvider TokenProvider
}

type UserRepository interface {
	Create(ctx context.Context, login, passwordHash string) (id int, err error)
	FindByLogin(ctx context.Context, login string) (id int, passwordHash string, err error)
}

type Hasher interface {
	Hash(string) (string, error)
	Compare(password, hash string) bool
}

type TokenProvider interface {
	GrantToken(ctx context.Context, userID int) (string, error)
}

func NewSignup(r UserRepository, h Hasher, p TokenProvider) *Signup {
	return &Signup{
		repository:    r,
		hasher:        h,
		tokenProvider: p,
	}
}

// Register создает нового пользователя в UserRepository и выдает ему авторизационный токен.
func (s *Signup) Register(ctx context.Context, login, password string) (string, error) {
	passwordHash, err := s.hasher.Hash(password)
	if err != nil {
		return "", err
	}

	id, err := s.repository.Create(ctx, login, passwordHash)
	if err != nil {
		return "", err
	}

	return s.tokenProvider.GrantToken(ctx, id)
}

// Login получает данные пользователя из UserRepository, проверяет совпадение хэша пароля
// и выдает новый авторизационный токен пользователю.
func (s *Signup) Login(ctx context.Context, login, password string) (string, error) {
	id, passwordHash, err := s.repository.FindByLogin(ctx, login)
	if err != nil {
		return "", inerr.ErrUserNotFound
	}

	if !s.hasher.Compare(password, passwordHash) {
		return "", inerr.ErrUserNotFound
	}

	return s.tokenProvider.GrantToken(ctx, id)
}
