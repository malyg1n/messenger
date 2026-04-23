package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/segmentio/kafka-go"

	_ "github.com/lib/pq"

	"message-service/internal/config"
	"message-service/internal/consumer"
	"message-service/internal/repository"
	"message-service/internal/service"
)

type App struct {
	Config       config.Config
	Logger       *slog.Logger
	DB           *sql.DB
	KafkaReader  *kafka.Reader
	KafkaConsume *consumer.KafkaConsumer
}

func Build() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	logger := newLogger(cfg.LogLevel)
	if cfg.LoadedEnvFile != "" {
		logger.Debug("dotenv loaded", "component", "bootstrap", "operation", "config.load_dotenv", "path", cfg.LoadedEnvFile)
	} else {
		logger.Debug("dotenv file not found, using process env only", "component", "bootstrap", "operation", "config.load_dotenv")
	}

	db, err := initDB(cfg)
	if err != nil {
		return nil, err
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: cfg.KafkaBrokers,
		Topic:   cfg.KafkaTopic,
		GroupID: cfg.KafkaGroupID,
	})

	messageRepo := repository.NewMessageRepository(db)
	messageSvc := service.NewMessageService(messageRepo)
	messageConsumer := consumer.NewKafkaConsumer(reader, messageSvc, logger, cfg.KafkaTopic)

	return &App{
		Config:       cfg,
		Logger:       logger,
		DB:           db,
		KafkaReader:  reader,
		KafkaConsume: messageConsumer,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	if err := a.KafkaConsume.Run(ctx); err != nil {
		return fmt.Errorf("run kafka consumer: %w", err)
	}

	return nil
}

func (a *App) Close() {
	if a.KafkaReader != nil {
		if err := a.KafkaReader.Close(); err != nil {
			a.Logger.Error("failed to close kafka reader", "component", "bootstrap", "operation", "close.kafka_reader", "error", err)
		}
	}
	if a.DB != nil {
		if err := a.DB.Close(); err != nil {
			a.Logger.Error("failed to close db", "component", "bootstrap", "operation", "close.db", "error", err)
		}
	}
}

func initDB(cfg config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return db, nil
}

func newLogger(level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}
