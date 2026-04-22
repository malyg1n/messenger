package bootstrap

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"api-service/internal/config"
	httpHandlers "api-service/internal/http/handlers"
	"api-service/internal/kafka"
	"api-service/internal/repository"
	"api-service/internal/service"
	_ "github.com/lib/pq"
)

type App struct {
	Config         config.Config
	Logger         *slog.Logger
	DB             *sql.DB
	KafkaProducer  kafka.Producer
	HTTPServer     *http.Server
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

	producer := kafka.NewWriterProducer(cfg.KafkaBrokers, cfg.KafkaClientID)

	userRepo := repository.NewUserRepository(db)
	chatRepo := repository.NewChatRepository(db)
	messageRepo := repository.NewMessageRepository(db)

	authSvc := service.NewAuthService(userRepo, producer, cfg.KafkaTopicUserRegistered, logger)
	userSvc := service.NewUserService(userRepo)
	chatSvc := service.NewChatService(chatRepo, producer, cfg.KafkaTopicChatCreated, logger)
	messageSvc := service.NewMessageService(messageRepo)

	handler := httpHandlers.New(authSvc, userSvc, chatSvc, messageSvc, logger)
	mux := http.NewServeMux()
	handler.Register(mux)

	finalHandler := httpHandlers.CORS(cfg.CORSAllowedOrigin, httpHandlers.Logging(logger, mux))
	srv := &http.Server{
		Addr:    ":" + cfg.APIPort,
		Handler: finalHandler,
	}

	return &App{
		Config:        cfg,
		Logger:        logger,
		DB:            db,
		KafkaProducer: producer,
		HTTPServer:    srv,
	}, nil
}

func (a *App) Close() {
	if a.KafkaProducer != nil {
		if err := a.KafkaProducer.Close(); err != nil {
			a.Logger.Error("failed to close kafka producer", "component", "bootstrap", "operation", "close.kafka", "error", err)
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
