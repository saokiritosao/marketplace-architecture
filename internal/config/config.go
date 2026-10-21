package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	HTTPAddr        string
	DatabaseURL     string
	JWTSecret       string
	JWTIssuer       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	OrderRateLimit  time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:    env("HTTP_ADDR", ":8080"),
		DatabaseURL: env("DATABASE_URL", "postgres://marketplace:marketplace@localhost:5432/marketplace?sslmode=disable"),
		JWTSecret:   env("JWT_SECRET", "local-development-secret-change-me"),
		JWTIssuer:   env("JWT_ISSUER", "marketplace-api"),
	}

	var err error
	cfg.AccessTokenTTL, err = duration("ACCESS_TOKEN_TTL", 20*time.Minute)
	if err != nil {
		return Config{}, err
	}
	cfg.RefreshTokenTTL, err = duration("REFRESH_TOKEN_TTL", 14*24*time.Hour)
	if err != nil {
		return Config{}, err
	}
	cfg.OrderRateLimit, err = duration("ORDER_RATE_LIMIT", time.Minute)
	if err != nil {
		return Config{}, err
	}

	if len(cfg.JWTSecret) < 24 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 24 characters")
	}

	return cfg, nil
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func duration(name string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	return parsed, nil
}
