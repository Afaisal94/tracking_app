package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Env  string
	Port string
	Host string

	DatabaseURL   string
	DBMaxOpenConns int
	DBMaxIdleConns int

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	JWTSecret       string
	JWTAccessExpiry time.Duration
	JWTRefreshExpiry time.Duration
	JWTIssuer       string

	WSIdleTimeout    time.Duration
	WSFlushInterval  time.Duration
	WSFlushBatchSize int

	RateLimitAuthRPS   int
	RateLimitAuthBurst int

	CORSAllowedOrigins []string
}

func Load() *Config {
	_ = godotenv.Load()

	c := &Config{
		Env:  env("APP_ENV", "development"),
		Port: env("APP_PORT", "8080"),
		Host: env("APP_HOST", "0.0.0.0"),

		DatabaseURL:      env("DATABASE_URL", "postgres://tracking:tracking_secret@localhost:5432/tracking_db?sslmode=disable"),
		DBMaxOpenConns:   envInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:   envInt("DB_MAX_IDLE_CONNS", 5),

		RedisAddr:     env("REDIS_ADDR", "localhost:6379"),
		RedisPassword: env("REDIS_PASSWORD", ""),
		RedisDB:       envInt("REDIS_DB", 0),

		JWTSecret:        env("JWT_SECRET", "change-this-super-secret-key-min-32-chars-please"),
		JWTAccessExpiry:  envDuration("JWT_ACCESS_EXPIRY", 15*time.Minute),
		JWTRefreshExpiry: envDuration("JWT_REFRESH_EXPIRY", 720*time.Hour),
		JWTIssuer:        env("JWT_ISSUER", "tracking-backend"),

		WSIdleTimeout:    envDuration("WS_IDLE_TIMEOUT", 15*time.Minute),
		WSFlushInterval:  envDuration("WS_FLUSH_INTERVAL", 10*time.Second),
		WSFlushBatchSize: envInt("WS_FLUSH_BATCH_SIZE", 50),

		RateLimitAuthRPS:   envInt("RATE_LIMIT_AUTH_RPS", 5),
		RateLimitAuthBurst: envInt("RATE_LIMIT_AUTH_BURST", 10),

		CORSAllowedOrigins: strings.Split(env("CORS_ALLOWED_ORIGINS", "*"), ","),
	}

	// Support REDIS_URL if provided
	if url := os.Getenv("REDIS_URL"); url != "" {
		// keep addr as is; go-redis can parse URL elsewhere, but we store addr
		// If URL contains redis://, extract host
		_ = url
	}

	return c
}

func (c *Config) Addr() string { return c.Host + ":" + c.Port }
func (c *Config) IsDev() bool  { return c.Env == "development" }

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
