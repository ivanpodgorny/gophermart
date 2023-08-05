package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilder_LoadEnv(t *testing.T) {
	var (
		serverAddress        = "localhost:8080"
		accrualSystemAddress = "localhost:8000"
		hmacKey              = "key"
		databaseURI          = "dsn"
		builder              = &Builder{
			parameters: &parameters{},
		}
	)

	require.NoError(t, os.Setenv("RUN_ADDRESS", serverAddress))
	require.NoError(t, os.Setenv("ACCRUAL_SYSTEM_ADDRESS", accrualSystemAddress))
	require.NoError(t, os.Setenv("HMAC_KEY", hmacKey))
	require.NoError(t, os.Setenv("DATABASE_URI", databaseURI))

	cfg, err := builder.LoadEnv().Build()
	require.NoError(t, err)
	assert.Equal(t, serverAddress, cfg.ServerAddress())
	assert.Equal(t, accrualSystemAddress, cfg.AccrualSystemAddress())
	assert.Equal(t, hmacKey, cfg.HMACKey())
	assert.Equal(t, databaseURI, cfg.DatabaseURI())
}

func TestBuilder_LoadFlags(t *testing.T) {
	var (
		serverAddress        = "localhost:8080"
		accrualSystemAddress = "localhost:8000"
		databaseURI          = "dsn"
		builder              = &Builder{
			parameters: &parameters{},
			arguments: []string{
				"-a", serverAddress,
				"-r", accrualSystemAddress,
				"-d", databaseURI,
			},
		}
	)

	cfg, err := builder.LoadFlags().Build()
	require.NoError(t, err)
	assert.Equal(t, serverAddress, cfg.ServerAddress())
	assert.Equal(t, accrualSystemAddress, cfg.AccrualSystemAddress())
	assert.Equal(t, databaseURI, cfg.DatabaseURI())
}
