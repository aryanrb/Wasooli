package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration, loaded from the environment.
type Config struct {
	Port        string
	DatabaseURL string
	Env         string
	JWTSecret   string
	FrontendURL string // SPA origin: CORS allow + post-login redirect target

	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
}

// Load reads config from the environment (a local .env is loaded if present,
// but real environment variables always win).
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:               getenv("PORT", "8080"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		Env:                getenv("RUNTIME_ENV", "dev"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		FrontendURL:        getenv("FRONTEND_URL", "http://localhost:5173"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  getenv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/v1/auth/google/callback"),
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	return cfg, nil
}

// SecureCookies reports whether auth cookies should be Secure (prod only).
func (c *Config) SecureCookies() bool { return c.Env != "dev" }

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
