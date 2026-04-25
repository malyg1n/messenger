package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

// Config описывает настройки ws-service из переменных окружения.
type Config struct {
	WSPort            string
	PostgresDSN       string
	KafkaBrokers      []string
	KafkaTopicMessage string
	KafkaGroupID      string
	LogLevel          slog.Level
	LoadedEnvFile     string
}

// Load загружает env, парсит LOG_LEVEL и валидирует обязательные значения.
func Load() (Config, error) {
	loadedEnvFile := loadDotEnv()

	cfg := Config{
		WSPort:            os.Getenv("WS_PORT"),
		PostgresDSN:       os.Getenv("POSTGRES_DSN"),
		KafkaTopicMessage: os.Getenv("KAFKA_TOPIC_MESSAGES"),
		KafkaGroupID:      os.Getenv("KAFKA_GROUP_ID"),
		LoadedEnvFile:     loadedEnvFile,
	}

	brokersRaw := os.Getenv("KAFKA_BROKERS")
	if brokersRaw != "" {
		parts := strings.Split(brokersRaw, ",")
		cfg.KafkaBrokers = make([]string, 0, len(parts))
		for _, part := range parts {
			broker := strings.TrimSpace(part)
			if broker != "" {
				cfg.KafkaBrokers = append(cfg.KafkaBrokers, broker)
			}
		}
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

// loadDotEnv пытается загрузить .env из стандартных путей разработки.
func loadDotEnv() string {
	candidates := []string{
		".env",
		filepath.Join("ws-service", ".env"),
	}
	for _, path := range candidates {
		if err := godotenv.Load(path); err == nil {
			return path
		}
	}
	return ""
}

// parseLogLevel преобразует строковое значение уровня в slog.Level.
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

// validate проверяет наличие обязательных env-переменных.
func validate(cfg Config) error {
	missing := make([]string, 0)
	if cfg.WSPort == "" {
		missing = append(missing, "WS_PORT")
	}
	if cfg.PostgresDSN == "" {
		missing = append(missing, "POSTGRES_DSN")
	}
	if len(cfg.KafkaBrokers) == 0 {
		missing = append(missing, "KAFKA_BROKERS")
	}
	if cfg.KafkaTopicMessage == "" {
		missing = append(missing, "KAFKA_TOPIC_MESSAGES")
	}
	if cfg.KafkaGroupID == "" {
		missing = append(missing, "KAFKA_GROUP_ID")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}
	return nil
}
