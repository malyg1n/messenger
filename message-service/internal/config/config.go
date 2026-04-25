package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

var (
	ErrInvalidConfig = errors.New("invalid config")
	ErrMissingEnv    = errors.New("missing required env vars")
)

// Config описывает настройки message-service, получаемые из окружения.
type Config struct {
	PostgresDSN        string
	KafkaBrokers       []string
	KafkaTopicIncoming string
	KafkaTopicSaved    string
	KafkaGroupID       string
	ProbePort          string
	LogLevel           slog.Level
	LoadedEnvFile      string
}

// Load загружает переменные окружения, парсит лог-уровень и валидирует конфиг.
func Load() (Config, error) {
	loadedEnvFile := loadDotEnv()

	cfg := Config{
		PostgresDSN:        os.Getenv("POSTGRES_DSN"),
		KafkaTopicIncoming: os.Getenv("KAFKA_TOPIC_MESSAGES_INCOMING"),
		KafkaTopicSaved:    os.Getenv("KAFKA_TOPIC_MESSAGES_SAVED"),
		KafkaGroupID:       os.Getenv("KAFKA_GROUP_ID"),
		ProbePort:          os.Getenv("PROBE_PORT"),
		LoadedEnvFile:      loadedEnvFile,
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
		return Config{}, fmt.Errorf("parse LOG_LEVEL: %w", err)
	}
	cfg.LogLevel = level

	if err := validate(cfg); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}

// loadDotEnv пытается загрузить .env из типовых путей.
func loadDotEnv() string {
	candidates := []string{
		".env",
		filepath.Join("message-service", ".env"),
	}
	for _, path := range candidates {
		if err := godotenv.Load(path); err == nil {
			return path
		}
	}

	return ""
}

// parseLogLevel переводит строковое значение LOG_LEVEL в slog.Level.
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
		return 0, fmt.Errorf("%w: LOG_LEVEL=%q", ErrInvalidConfig, value)
	}
}

// validate проверяет присутствие обязательных переменных окружения.
func validate(cfg Config) error {
	missing := make([]string, 0)
	if cfg.PostgresDSN == "" {
		missing = append(missing, "POSTGRES_DSN")
	}
	if len(cfg.KafkaBrokers) == 0 {
		missing = append(missing, "KAFKA_BROKERS")
	}
	if cfg.KafkaTopicIncoming == "" {
		missing = append(missing, "KAFKA_TOPIC_MESSAGES_INCOMING")
	}
	if cfg.KafkaTopicSaved == "" {
		missing = append(missing, "KAFKA_TOPIC_MESSAGES_SAVED")
	}
	if cfg.KafkaGroupID == "" {
		missing = append(missing, "KAFKA_GROUP_ID")
	}
	if cfg.ProbePort == "" {
		missing = append(missing, "PROBE_PORT")
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: %s", ErrMissingEnv, strings.Join(missing, ", "))
	}

	return nil
}
