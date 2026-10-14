// Package config loads application configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
	EnvTest        = "test"
)

type Config struct {
	Env  string
	HTTP HTTPConfig
	Log  LogConfig
	DB   DBConfig
	CORS CORSConfig
}

type HTTPConfig struct {
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	// TrustedProxies are the CIDRs/IPs whose X-Forwarded-For header is trusted.
	// Empty means "trust nobody": ClientIP() is the direct peer address.
	TrustedProxies []string
}

type LogConfig struct {
	Level  string // debug | info | warn | error
	Format string // text | json
}

type DBConfig struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

type CORSConfig struct {
	AllowedOrigins []string
}

func (c Config) IsProduction() bool { return c.Env == EnvProduction }

// Load reads configuration from the process environment and validates it.
// All problems are collected and returned together.
func Load() (Config, error) {
	l := &loader{}

	env := l.str("APP_ENV", EnvDevelopment)
	defaultFormat := "json"
	if env == EnvDevelopment {
		defaultFormat = "text"
	}

	cfg := Config{
		Env: env,
		HTTP: HTTPConfig{
			Port:            l.int("HTTP_PORT", 8080),
			ReadTimeout:     l.dur("HTTP_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:    l.dur("HTTP_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:     l.dur("HTTP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: l.dur("HTTP_SHUTDOWN_TIMEOUT", 15*time.Second),
			TrustedProxies:  l.list("TRUSTED_PROXIES"),
		},
		Log: LogConfig{
			Level:  strings.ToLower(l.str("LOG_LEVEL", "info")),
			Format: strings.ToLower(l.str("LOG_FORMAT", defaultFormat)),
		},
		DB: DBConfig{
			URL:             l.str("DATABASE_URL", ""),
			MaxConns:        int32(l.int("DB_MAX_CONNS", 10)),
			MinConns:        int32(l.int("DB_MIN_CONNS", 2)),
			MaxConnLifetime: l.dur("DB_MAX_CONN_LIFETIME", time.Hour),
			MaxConnIdleTime: l.dur("DB_MAX_CONN_IDLE_TIME", 30*time.Minute),
		},
		CORS: CORSConfig{
			AllowedOrigins: l.list("CORS_ALLOWED_ORIGINS"),
		},
	}

	switch cfg.Env {
	case EnvDevelopment, EnvProduction, EnvTest:
	default:
		l.errs = append(l.errs, fmt.Errorf("APP_ENV must be development, production or test, got %q", cfg.Env))
	}
	if cfg.DB.URL == "" {
		l.errs = append(l.errs, errors.New("DATABASE_URL is required"))
	}
	if cfg.HTTP.Port < 1 || cfg.HTTP.Port > 65535 {
		l.errs = append(l.errs, fmt.Errorf("HTTP_PORT out of range: %d", cfg.HTTP.Port))
	}
	if cfg.DB.MinConns > cfg.DB.MaxConns {
		l.errs = append(l.errs, errors.New("DB_MIN_CONNS must be <= DB_MAX_CONNS"))
	}

	if err := errors.Join(l.errs...); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

type loader struct{ errs []error }

func (l *loader) str(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func (l *loader) int(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		l.errs = append(l.errs, fmt.Errorf("%s: invalid integer %q", key, v))
		return def
	}
	return n
}

func (l *loader) dur(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		l.errs = append(l.errs, fmt.Errorf("%s: invalid duration %q", key, v))
		return def
	}
	return d
}

// list parses a comma-separated variable, trimming blanks.
func (l *loader) list(key string) []string {
	var out []string
	for _, p := range strings.Split(os.Getenv(key), ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
