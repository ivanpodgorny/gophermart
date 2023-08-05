package config

import (
	"flag"
	"github.com/caarlos0/env/v8"
	"os"
)

type Config struct {
	parameters *parameters
}

type Builder struct {
	parameters *parameters
	err        error
	arguments  []string
}

type parameters struct {
	ServerAddress        string `env:"RUN_ADDRESS"`
	HMACKey              string `env:"HMAC_KEY"`
	DatabaseURI          string `env:"DATABASE_URI"`
	AccrualSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
}

const (
	defaultServerAddress = "localhost:8080"
)

func NewBuilder() *Builder {
	return &Builder{
		arguments: os.Args[1:],
		parameters: &parameters{
			ServerAddress: defaultServerAddress,
		},
	}
}

func (b *Builder) SetDefaultServerAddress(addr string) *Builder {
	b.parameters.ServerAddress = addr

	return b
}

func (b *Builder) LoadEnv() *Builder {
	if err := env.Parse(b.parameters); err != nil {
		b.err = err
	}

	return b
}

func (b *Builder) LoadFlags() *Builder {
	flag.StringVar(&b.parameters.ServerAddress, "a", b.parameters.ServerAddress, "адрес и порт запуска сервиса HTTP-сервера")
	flag.StringVar(&b.parameters.DatabaseURI, "d", "", "адрес подключения к PostgreSQL")
	flag.StringVar(&b.parameters.AccrualSystemAddress, "r", "", "адрес системы расчёта начислений")

	err := flag.CommandLine.Parse(b.arguments)
	if err != nil {
		b.err = err
	}

	return b
}

func (b *Builder) Build() (*Config, error) {
	return &Config{b.parameters}, b.err
}

func (c *Config) ServerAddress() string {
	return c.parameters.ServerAddress
}

func (c *Config) HMACKey() string {
	return c.parameters.HMACKey
}

func (c *Config) DatabaseURI() string {
	return c.parameters.DatabaseURI
}

func (c *Config) AccrualSystemAddress() string {
	return c.parameters.AccrualSystemAddress
}
