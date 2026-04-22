package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"ws-service/internal/broker"
	"ws-service/internal/config"
	"ws-service/internal/store"
	"ws-service/internal/ws"

	_ "github.com/lib/pq"
)

type App struct {
	Config        config.Config
	Logger        *slog.Logger
	DB            *sql.DB
	KafkaProducer broker.Producer
	KafkaConsumer broker.Consumer
	HTTPServer    *http.Server
	WSConsumer    *ws.Consumer
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

	producer := broker.NewWriterProducer(cfg.KafkaBrokers, cfg.KafkaTopicMessage)
	consumer := broker.NewReaderConsumer(cfg.KafkaBrokers, cfg.KafkaTopicMessage, cfg.KafkaGroupID)

	participantStore := store.NewParticipantStore(db)
	hub := ws.NewHub(logger)
	handler := ws.NewHandler(producer, hub, logger)
	wsConsumer := ws.NewConsumer(consumer, participantStore, hub, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler.HandleWS)

	server := &http.Server{
		Addr:    ":" + cfg.WSPort,
		Handler: mux,
	}

	return &App{
		Config:        cfg,
		Logger:        logger,
		DB:            db,
		KafkaProducer: producer,
		KafkaConsumer: consumer,
		HTTPServer:    server,
		WSConsumer:    wsConsumer,
	}, nil
}

func (a *App) RunConsumer(ctx context.Context) {
	a.WSConsumer.Run(ctx)
}

func (a *App) Close() {
	if a.KafkaProducer != nil {
		if err := a.KafkaProducer.Close(); err != nil {
			a.Logger.Error("failed to close kafka producer", "component", "bootstrap", "operation", "close.kafka_producer", "error", err)
		}
	}
	if a.KafkaConsumer != nil {
		if err := a.KafkaConsumer.Close(); err != nil {
			a.Logger.Error("failed to close kafka consumer", "component", "bootstrap", "operation", "close.kafka_consumer", "error", err)
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
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return db, nil
}

func newLogger(level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	return slog.New(handler)
}
