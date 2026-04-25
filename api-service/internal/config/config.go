package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

// Config описывает обязательные настройки api-service из окружения.
type Config struct {
	APIPort           string
	PostgresDSN       string
	CORSAllowedOrigin string
	LogLevel          slog.Level
	LoadedEnvFile     string
}

// Load загружает переменные окружения, парсит лог-уровень и валидирует конфиг.
func Load() (Config, error) {
	loadedEnvFile := loadDotEnv()

	cfg := Config{
		APIPort:           os.Getenv("API_PORT"),
		PostgresDSN:       os.Getenv("POSTGRES_DSN"),
		CORSAllowedOrigin: os.Getenv("CORS_ALLOWED_ORIGIN"),
		LoadedEnvFile:     loadedEnvFile,
	}

	level, err := parseLogLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		return Config{}, err
	}
	cfg.LogLevel = level

	if err := validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// loadDotEnv пытается подгрузить .env из стандартных путей разработки.
func loadDotEnv() string {
	candidates := []string{
		".env",
		filepath.Join("api-service", ".env"),
	}
	for _, path := range candidates {
		if err := godotenv.Load(path); err == nil {
			return path
		}
	}
	return ""
}

// parseLogLevel преобразует строку уровня логирования в slog.Level.
func parseLogLevel(value string) (slog.Level, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid LOG_LEVEL: %q", value)
	}
}

// validate проверяет наличие обязательных переменных окружения.
func validate(cfg Config) error {
	missing := make([]string, 0)
	if cfg.APIPort == "" {
		missing = append(missing, "API_PORT")
	}
	if cfg.PostgresDSN == "" {
		missing = append(missing, "POSTGRES_DSN")
	}
	if cfg.CORSAllowedOrigin == "" {
		missing = append(missing, "CORS_ALLOWED_ORIGIN")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}
	return nil
}
