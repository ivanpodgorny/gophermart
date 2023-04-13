package security

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHMACSigner(t *testing.T) {
	var (
		token   = "token"
		invalid = "invalid"
		signer  = NewHMACSigner("")
		signed  = signer.Sign(token)
	)

	parsed, _ := signer.Parse(signed)
	assert.Equal(t, token, parsed, "успешная проверка токена")

	_, err := signer.Parse(invalid)
	assert.Error(t, err, "неуспешная проверка токена")
}
