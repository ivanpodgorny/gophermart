package security

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"golang.org/x/crypto/argon2"
	"strings"
)

// ArgonHasher реализует service.Hasher с использованием Argon2.
type ArgonHasher struct {
	cfg *HashConfig
}

type HashConfig struct {
	Time    uint32
	Memory  uint32
	Threads uint8
	KeyLen  uint32
}

func NewArgonHasher(cfg *HashConfig) *ArgonHasher {
	return &ArgonHasher{cfg: cfg}
}

func DefaultHashConfig() *HashConfig {
	return &HashConfig{
		Time:    1,
		Memory:  64 * 1024,
		Threads: 4,
		KeyLen:  32,
	}
}

func (h *ArgonHasher) Hash(str string) (string, error) {
	salt, err := RandomBytes(16)
	if err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(str), salt, h.cfg.Time, h.cfg.Memory, h.cfg.Threads, h.cfg.KeyLen)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		h.cfg.Memory,
		h.cfg.Time,
		h.cfg.Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func (h *ArgonHasher) Compare(str, hash string) bool {
	parts := strings.Split(hash, "$")
	if len(parts) < 5 {
		return false
	}

	c := &HashConfig{}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &c.Memory, &c.Time, &c.Threads); err != nil {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	comparisonHash := argon2.IDKey([]byte(str), salt, c.Time, c.Memory, c.Threads, uint32(len(decodedHash)))

	return subtle.ConstantTimeCompare(decodedHash, comparisonHash) == 1
}
