package middleware

import (
	"net/http"
)

type Authenticator interface {
	Authenticate(signed string, r *http.Request) (*http.Request, error)
}

// Authenticate возвращает middleware для поверки токена пользователя.
func Authenticate(a Authenticator) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			signed := r.Header.Get("Authorization")
			r, err := a.Authenticate(signed, r)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
