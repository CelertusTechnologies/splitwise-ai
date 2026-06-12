package config

import (
	"os"
	"strconv"
	"time"
)

const (
	DriverPostgres = "postgres"
	DriverSQLite   = "sqlite"
)

type Config struct {
	AppName          string
	Env              string
	Port             string
	DBDriver         string
	DatabaseURL      string
	RedisAddr        string
	RedisPassword    string
	FrontendURL      string
	JWTAccessSecret  string
	JWTRefreshSecret string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
}

func Load() Config {
	accessMinutes := getInt("JWT_ACCESS_TTL_MINUTES", 15)
	refreshDays := getInt("JWT_REFRESH_TTL_DAYS", 30)

	driver := getString("DB_DRIVER", DriverPostgres)

	// In SQLite dev mode the DATABASE_URL is a file path; only fall back to the
	// Postgres DSN default when we are actually running against Postgres.
	defaultDSN := "postgres://nivra:nivra@localhost:5432/nivra?sslmode=disable"
	if driver == DriverSQLite {
		defaultDSN = "nivra-dev.db"
	}

	return Config{
		AppName:          getString("APP_NAME", "Nivra"),
		Env:              getString("APP_ENV", "development"),
		Port:             getString("BACKEND_PORT", "8080"),
		DBDriver:         driver,
		DatabaseURL:      getString("DATABASE_URL", defaultDSN),
		RedisAddr:        getString("REDIS_ADDR", "localhost:6379"),
		RedisPassword:    getString("REDIS_PASSWORD", ""),
		FrontendURL:      getString("FRONTEND_URL", "http://localhost:3000"),
		JWTAccessSecret:  getString("JWT_ACCESS_SECRET", "local-dev-access-secret-change-in-production"),
		JWTRefreshSecret: getString("JWT_REFRESH_SECRET", "local-dev-refresh-secret-change-in-production"),
		AccessTokenTTL:   time.Duration(accessMinutes) * time.Minute,
		RefreshTokenTTL:  time.Duration(refreshDays) * 24 * time.Hour,
	}
}

// IsProduction reports whether the app is running in the production environment.
func (c Config) IsProduction() bool {
	return c.Env == "production"
}

func getString(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}
