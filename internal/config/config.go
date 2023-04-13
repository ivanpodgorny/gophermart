package config

import (
	"flag"
	"github.com/caarlos0/env/v8"
)

type Config interface {
	ServerAddress() string
	HMACKey() string
	DatabaseURI() string
	AccrualSystemAddress() string
}

type Builder struct {
	parameters *parameters
	err        error
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
	b.err = env.Parse(b.parameters)

	return b
}

func (b *Builder) LoadFlags() *Builder {
	flag.StringVar(&b.parameters.ServerAddress, "a", b.parameters.ServerAddress, "адрес и порт запуска сервиса HTTP-сервера")
	flag.StringVar(&b.parameters.DatabaseURI, "d", "", "адрес подключения к PostgreSQL")
	flag.StringVar(&b.parameters.AccrualSystemAddress, "r", "", "адрес системы расчёта начислений")
	flag.Parse()

	return b
}

func (b *Builder) Build() (Config, error) {
	return b, b.err
}

func (b *Builder) ServerAddress() string {
	return b.parameters.ServerAddress
}

func (b *Builder) HMACKey() string {
	return b.parameters.HMACKey
}

func (b *Builder) DatabaseURI() string {
	return b.parameters.DatabaseURI
}

func (b *Builder) AccrualSystemAddress() string {
	return b.parameters.AccrualSystemAddress
}
