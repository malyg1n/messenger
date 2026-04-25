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

	"ws-service/internal/broker"
	"ws-service/internal/cache"
	"ws-service/internal/config"
	"ws-service/internal/service"
	"ws-service/internal/store"
	"ws-service/internal/ws"

	_ "github.com/lib/pq"
)

// App хранит инфраструктурные зависимости ws-service.
type App struct {
	Config        config.Config
	Logger        *slog.Logger
	DB            *sql.DB
	KafkaProducer broker.Producer
	KafkaConsumer broker.Consumer
	HTTPServer    *http.Server
	WSConsumer    *ws.Consumer
}

// Build собирает все зависимости ws-service:
// config -> logger -> db -> kafka -> сервисы домена -> http-обработчики.
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

	// Сервис участников нужен Kafka-консьюмеру, чтобы определить,
	// кому доставлять каждое сообщение чата в реальном времени.
	participantsCache := cache.NewParticipantsCache()
	participantStore := store.NewParticipantStore(db)
	participantsService := service.NewParticipantsService(participantsCache, participantStore, logger)
	hub := ws.NewHub(logger)
	handler := ws.NewHandler(producer, hub, logger)
	wsConsumer := ws.NewConsumer(consumer, participantsService, hub, logger)

	// Сервис публикует health/readiness-пробы и websocket-эндпоинт.
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/ready", readyHandler(db, producer, consumer))
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

// RunConsumer запускает фоновую обработку сообщений из Kafka.
func (a *App) RunConsumer(ctx context.Context) {
	a.WSConsumer.Run(ctx)
}

// Close закрывает producer/consumer Kafka и соединение с БД.
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

// initDB открывает подключение к Postgres и проверяет его доступность.
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

// newLogger создает JSON-логгер.
func newLogger(level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
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

// readyHandler проверяет доступность БД и Kafka producer/consumer.
func readyHandler(db *sql.DB, producer broker.Producer, consumer broker.Consumer) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		// Проверки готовности должны быть быстрыми, чтобы не зависали пробы оркестратора.
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			writeProbeResponse(w, http.StatusServiceUnavailable, probeResponse{
				Status: "not_ready",
				Error:  fmt.Sprintf("db ping failed: %v", err),
			})
			return
		}

		if err := producer.Ready(ctx); err != nil {
			probeErr := fmt.Sprintf("kafka producer unavailable: %v", err)
			switch {
			case errors.Is(err, broker.ErrNoKafkaBrokers):
				probeErr = "kafka producer unavailable: no brokers configured"
			case errors.Is(err, broker.ErrKafkaUnavailable):
				probeErr = "kafka producer unavailable: brokers are unreachable"
			}
			writeProbeResponse(w, http.StatusServiceUnavailable, probeResponse{
				Status: "not_ready",
				Error:  probeErr,
			})
			return
		}

		if err := consumer.Ready(ctx); err != nil {
			probeErr := fmt.Sprintf("kafka consumer unavailable: %v", err)
			switch {
			case errors.Is(err, broker.ErrNoKafkaBrokers):
				probeErr = "kafka consumer unavailable: no brokers configured"
			case errors.Is(err, broker.ErrKafkaUnavailable):
				probeErr = "kafka consumer unavailable: brokers are unreachable"
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

// writeProbeResponse формирует JSON-ответ probe-эндпоинтов.
func writeProbeResponse(w http.ResponseWriter, statusCode int, payload probeResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, `{"status":"error","error":"encode response failed"}`, http.StatusInternalServerError)
	}
}
