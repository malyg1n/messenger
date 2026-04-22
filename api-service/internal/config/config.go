package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	APIPort                 string
	PostgresDSN             string
	CORSAllowedOrigin       string
	KafkaBrokers            []string
	KafkaClientID           string
	KafkaTopicUserRegistered string
	KafkaTopicChatCreated   string
	LogLevel                slog.Level
	LoadedEnvFile           string
}

func Load() (Config, error) {
	loadedEnvFile := loadDotEnv()

	cfg := Config{
		APIPort:                 os.Getenv("API_PORT"),
		PostgresDSN:             os.Getenv("POSTGRES_DSN"),
		CORSAllowedOrigin:       os.Getenv("CORS_ALLOWED_ORIGIN"),
		KafkaClientID:           os.Getenv("KAFKA_CLIENT_ID"),
		KafkaTopicUserRegistered: os.Getenv("KAFKA_TOPIC_USER_REGISTERED"),
		KafkaTopicChatCreated:   os.Getenv("KAFKA_TOPIC_CHAT_CREATED"),
		LoadedEnvFile:           loadedEnvFile,
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
	if len(cfg.KafkaBrokers) == 0 {
		missing = append(missing, "KAFKA_BROKERS")
	}
	if cfg.KafkaClientID == "" {
		missing = append(missing, "KAFKA_CLIENT_ID")
	}
	if cfg.KafkaTopicUserRegistered == "" {
		missing = append(missing, "KAFKA_TOPIC_USER_REGISTERED")
	}
	if cfg.KafkaTopicChatCreated == "" {
		missing = append(missing, "KAFKA_TOPIC_CHAT_CREATED")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}
	return nil
}
