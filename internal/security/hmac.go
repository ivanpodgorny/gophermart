package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

// HMACSigner реализует Signer с использованием HMAC.
type HMACSigner struct {
	key string
}

var ErrIncorrectHMACSignature = errors.New("incorrect signature")

func NewHMACSigner(key string) *HMACSigner {
	return &HMACSigner{key: key}
}

func (s *HMACSigner) Sign(token string) string {
	return token + "/" + hex.EncodeToString(s.signHMAC([]byte(token), s.key))
}

func (s *HMACSigner) Parse(signed string) (string, error) {
	parts := strings.Split(signed, "/")
	if len(parts) < 2 {
		return "", ErrIncorrectHMACSignature
	}

	token := parts[0]
	hmacSign, err := hex.DecodeString(parts[1])
	if err != nil {
		return "", err
	}

	if s.validateHMAC([]byte(token), hmacSign, s.key) {
		return token, nil
	}

	return "", ErrIncorrectHMACSignature
}

func (s *HMACSigner) signHMAC(data []byte, key string) []byte {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)

	return h.Sum(nil)
}

func (s *HMACSigner) validateHMAC(data, sign []byte, key string) bool {
	return hmac.Equal(s.signHMAC(data, key), sign)
}
