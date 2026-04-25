package bootstrap

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"api-service/internal/config"
	httpHandlers "api-service/internal/http/handlers"
	"api-service/internal/repository"
	"api-service/internal/service"

	_ "github.com/lib/pq"
)

// App хранит инфраструктурные зависимости запущенного api-service.
type App struct {
	Config     config.Config
	Logger     *slog.Logger
	DB         *sql.DB
	HTTPServer *http.Server
}

// Build собирает объект приложения и связывает конфиг, БД, сервисы и HTTP-роуты.
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

	userRepo := repository.NewUserRepository(db)
	chatRepo := repository.NewChatRepository(db)
	messageRepo := repository.NewMessageRepository(db)

	authSvc := service.NewAuthService(userRepo, logger)
	userSvc := service.NewUserService(userRepo)
	chatSvc := service.NewChatService(chatRepo, logger)
	messageSvc := service.NewMessageService(messageRepo)

	handler := httpHandlers.New(authSvc, userSvc, chatSvc, messageSvc, logger)
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/ready", readyHandler(db))
	handler.Register(mux)

	finalHandler := httpHandlers.CORS(cfg.CORSAllowedOrigin, httpHandlers.Logging(logger, mux))
	srv := &http.Server{
		Addr:    ":" + cfg.APIPort,
		Handler: finalHandler,
	}

	return &App{
		Config:     cfg,
		Logger:     logger,
		DB:         db,
		HTTPServer: srv,
	}, nil
}

// Close освобождает внешние ресурсы приложения.
func (a *App) Close() {
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

// newLogger создает JSON-логгер с уровнем из конфигурации.
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

// healthHandler сообщает, что процесс жив и принимает запросы.
func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeProbeResponse(w, http.StatusOK, probeResponse{Status: "ok"})
}

// readyHandler проверяет готовность приложения обслуживать трафик.
func readyHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			writeProbeResponse(w, http.StatusServiceUnavailable, probeResponse{
				Status: "not_ready",
				Error:  fmt.Sprintf("db ping failed: %v", err),
			})
			return
		}

		writeProbeResponse(w, http.StatusOK, probeResponse{Status: "ready"})
	}
}

// writeProbeResponse унифицирует JSON-ответы health/readiness эндпоинтов.
func writeProbeResponse(w http.ResponseWriter, statusCode int, payload probeResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, `{"status":"error","error":"encode response failed"}`, http.StatusInternalServerError)
	}
}
