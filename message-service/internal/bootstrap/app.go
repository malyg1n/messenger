package bootstrap

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/segmentio/kafka-go"

	_ "github.com/lib/pq"

	"message-service/internal/broker"
	"message-service/internal/config"
	"message-service/internal/consumer"
	"message-service/internal/repository"
	"message-service/internal/service"
)

var (
	ErrNoKafkaBrokers   = errors.New("no kafka brokers configured")
	ErrKafkaUnavailable = errors.New("kafka brokers are unavailable")
)

const (
	httpShutdownTimeout = 5 * time.Second
	consumerStopTimeout = 30 * time.Second
)

// App хранит инфраструктурные зависимости message-service.
type App struct {
	Config       config.Config
	Logger       *slog.Logger
	DB           *sql.DB
	KafkaReader  *kafka.Reader
	KafkaWriter  *broker.Producer
	KafkaConsume *consumer.KafkaConsumer
	HTTPServer   *http.Server
}

// Build собирает конфиг, инфраструктуру Kafka/Postgres и runtime-объекты сервиса.
func Build(ctx context.Context) (*App, error) {
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

	db, err := initDB(ctx, cfg)
	if err != nil {
		return nil, err
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: cfg.KafkaBrokers,
		Topic:   cfg.KafkaTopicIncoming,
		GroupID: cfg.KafkaGroupID,
	})
	writer := broker.NewProducer(cfg.KafkaBrokers, cfg.KafkaTopicSaved)

	messageRepo := repository.NewMessageRepository(db)
	messageSvc := service.NewMessageService(messageRepo, writer)
	messageConsumer := consumer.NewKafkaConsumer(reader, messageSvc, logger, cfg.KafkaTopicIncoming)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/ready", readyHandler(db, cfg.KafkaBrokers))

	httpServer := &http.Server{
		Addr:    ":" + cfg.ProbePort,
		Handler: mux,
	}

	return &App{
		Config:       cfg,
		Logger:       logger,
		DB:           db,
		KafkaReader:  reader,
		KafkaWriter:  writer,
		KafkaConsume: messageConsumer,
		HTTPServer:   httpServer,
	}, nil
}

// Run параллельно запускает probe-сервер и Kafka-consumer, завершаясь по контексту.
func (a *App) Run(ctx context.Context) error {
	runCtx, stopConsumer := context.WithCancel(ctx)
	defer stopConsumer()

	serverErrCh := make(chan error, 1)
	go func() {
		if err := a.HTTPServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- fmt.Errorf("run http probe server: %w", err)
			return
		}
		serverErrCh <- nil
	}()

	consumerErrCh := make(chan error, 1)
	go func() {
		if err := a.KafkaConsume.Run(runCtx); err != nil {
			consumerErrCh <- fmt.Errorf("run kafka consumer: %w", err)
			return
		}
		consumerErrCh <- nil
	}()

	shutdownHTTP := func() error {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), httpShutdownTimeout)
		defer cancel()
		if err := a.HTTPServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("shutdown http probe server: %w", err)
		}
		return nil
	}

	waitConsumer := func() error {
		select {
		case err := <-consumerErrCh:
			return err
		case <-time.After(consumerStopTimeout):
			a.Logger.Warn(
				"kafka consumer did not stop in time; closing reader in Close may unblock it",
				"component", "bootstrap",
				"operation", "run.wait_consumer",
				"timeout", consumerStopTimeout.String(),
			)
			return fmt.Errorf("kafka consumer stop timeout after %s", consumerStopTimeout)
		}
	}

	select {
	case err := <-serverErrCh:
		stopConsumer()
		if werr := waitConsumer(); werr != nil {
			a.Logger.Error("consumer after http server error", "component", "bootstrap", "operation", "run.after_server_err", "error", werr)
			if err != nil {
				return errors.Join(err, werr)
			}
			return werr
		}
		return err

	case err := <-consumerErrCh:
		if shutdownErr := shutdownHTTP(); shutdownErr != nil {
			if err != nil {
				return errors.Join(err, shutdownErr)
			}
			return shutdownErr
		}
		return err

	case <-ctx.Done():
		stopConsumer()
		consumerErr := waitConsumer()
		if shutdownErr := shutdownHTTP(); shutdownErr != nil {
			if consumerErr != nil {
				return errors.Join(consumerErr, shutdownErr)
			}
			return shutdownErr
		}
		return consumerErr
	}
}

// Close корректно закрывает HTTP-сервер, Kafka reader и соединение с БД.
func (a *App) Close(ctx context.Context) {
	if a.HTTPServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), httpShutdownTimeout)
		defer cancel()
		if err := a.HTTPServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.Logger.Error("failed to close http server", "component", "bootstrap", "operation", "close.http_server", "error", err)
		}
	}
	if a.KafkaReader != nil {
		if err := a.KafkaReader.Close(); err != nil {
			a.Logger.Error("failed to close kafka reader", "component", "bootstrap", "operation", "close.kafka_reader", "error", err)
		}
	}
	if a.KafkaWriter != nil {
		if err := a.KafkaWriter.Close(); err != nil {
			a.Logger.Error("failed to close kafka writer", "component", "bootstrap", "operation", "close.kafka_writer", "error", err)
		}
	}
	if a.DB != nil {
		if err := a.DB.Close(); err != nil {
			a.Logger.Error("failed to close db", "component", "bootstrap", "operation", "close.db", "error", err)
		}
	}
}

// initDB открывает подключение к Postgres и проверяет доступность БД.
func initDB(ctx context.Context, cfg config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return db, nil
}

// newLogger создает JSON-логгер.
func newLogger(level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}

type probeResponse struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// healthHandler сообщает, что процесс жив.
func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeProbeResponse(w, http.StatusOK, probeResponse{Status: "ok"})
}

// readyHandler проверяет доступность БД и Kafka-брокеров.
func readyHandler(db *sql.DB, brokers []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			writeProbeResponse(w, http.StatusServiceUnavailable, probeResponse{
				Status: "not_ready",
				Error:  fmt.Sprintf("db ping failed: %v", err),
			})
			return
		}

		if err := checkKafkaBrokers(ctx, brokers); err != nil {
			probeErr := fmt.Sprintf("kafka unavailable: %v", err)
			switch {
			case errors.Is(err, ErrNoKafkaBrokers):
				probeErr = "kafka unavailable: no brokers configured"
			case errors.Is(err, ErrKafkaUnavailable):
				probeErr = "kafka unavailable: brokers are unreachable"
			}
			writeProbeResponse(w, http.StatusServiceUnavailable, probeResponse{
				Status: "not_ready",
				Error:  probeErr,
			})
			return
		}

		writeProbeResponse(w, http.StatusOK, probeResponse{Status: "ready"})
	}
}

// checkKafkaBrokers пытается подключиться хотя бы к одному брокеру Kafka.
func checkKafkaBrokers(ctx context.Context, brokers []string) error {
	if len(brokers) == 0 {
		return ErrNoKafkaBrokers
	}

	var lastErr error
	for _, brokerAddr := range brokers {
		conn, err := kafka.DialContext(ctx, "tcp", brokerAddr)
		if err != nil {
			lastErr = err
			continue
		}
		_ = conn.Close()
		return nil
	}

	if lastErr != nil {
		return fmt.Errorf("%w: %w", ErrKafkaUnavailable, lastErr)
	}
	return ErrKafkaUnavailable
}

// writeProbeResponse формирует JSON-ответ probe-эндпоинтов.
func writeProbeResponse(w http.ResponseWriter, statusCode int, payload probeResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, `{"status":"error","error":"encode response failed"}`, http.StatusInternalServerError)
	}
}
