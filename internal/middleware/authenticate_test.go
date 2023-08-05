package middleware

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

type AuthenticatorMock struct {
	mock.Mock
}

func (m *AuthenticatorMock) Authenticate(signed string, r *http.Request) (*http.Request, error) {
	args := m.Called(signed, r)

	return r, args.Error(1)
}

func TestAuthenticate(t *testing.T) {
	var (
		r             = chi.NewRouter()
		path          = "/"
		token         = "token"
		invalidToken  = "invalidToken"
		authenticator = &AuthenticatorMock{}
	)

	authenticator.
		On("Authenticate", token, mock.AnythingOfType("*http.Request")).
		Return(mock.AnythingOfType("*http.Request"), nil).
		Once()
	authenticator.
		On("Authenticate", invalidToken, mock.AnythingOfType("*http.Request")).
		Return(mock.AnythingOfType("*http.Request"), errors.New("")).
		Once()
	r.Use(Authenticate(authenticator))
	r.Post(path, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	tests := []struct {
		name           string
		token          string
		wantStatusCode int
	}{
		{
			name:           "успешная проверка токена",
			token:          token,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "невалидный токен",
			token:          invalidToken,
			wantStatusCode: http.StatusUnauthorized,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, ts.URL+path, nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", tt.token)
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())
			assert.Equal(t, tt.wantStatusCode, resp.StatusCode)
		})
	}
	authenticator.AssertExpectations(t)
}
