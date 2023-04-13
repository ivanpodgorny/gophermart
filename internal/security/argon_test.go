package security

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestArgonHasher(t *testing.T) {
	var (
		password      = "password"
		wrongPassword = "wrongPassword"
		hasher        = NewArgonHasher(DefaultHashConfig())
	)

	hash, err := hasher.Hash(password)
	assert.NoError(t, err, "создание хэша")

	assert.True(t, hasher.Compare(password, hash), "успешная проверка хэша")
	assert.False(t, hasher.Compare(wrongPassword, hash), "неуспешная проверка хэша")
}
