package security

import (
	"crypto/rand"
	"encoding/base64"
)

func RandomBytes(size int) ([]byte, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}

	return b, nil
}

func RandomString(size int) (string, error) {
	b, err := RandomBytes(size)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b)[:size], nil
}
