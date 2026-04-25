package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ws-service/internal/bootstrap"
)

// main — точка входа ws-service: запускает websocket-сервер и Kafka-consumer.
func main() {
	app, err := bootstrap.Build()
	if err != nil {
		slog.Error("bootstrap failed", "component", "cmd.ws", "operation", "build", "error", err)
		os.Exit(1)
	}
	defer app.Shutdown()

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go app.RunConsumer(rootCtx)

	serverErrCh := make(chan error, 1)
	go func() {
		if err := app.HTTPServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
			return
		}
		serverErrCh <- nil
	}()

	app.Logger.Info("ws started", "component", "cmd.ws", "operation", "start_server", "addr", app.HTTPServer.Addr)

	select {
	case err := <-serverErrCh:
		if err != nil {
			app.Logger.Error("server stopped with error", "component", "cmd.ws", "operation", "listen_and_serve", "error", err)
			app.Shutdown()
			os.Exit(1)
		}
		app.Logger.Info("server stopped", "component", "cmd.ws", "operation", "listen_and_serve")
	case <-rootCtx.Done():
		app.Logger.Info("shutdown signal received", "component", "cmd.ws", "operation", "signal", "reason", rootCtx.Err())

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		shutdownErr := app.HTTPServer.Shutdown(shutdownCtx)
		cancel()
		if shutdownErr != nil {
			app.Logger.Error("graceful http shutdown failed", "component", "cmd.ws", "operation", "shutdown", "error", shutdownErr)
			app.Shutdown()
			os.Exit(1)
		}

		if err := <-serverErrCh; err != nil {
			app.Logger.Error("server after shutdown", "component", "cmd.ws", "operation", "listen_and_serve", "error", err)
			app.Shutdown()
			os.Exit(1)
		}

		app.Logger.Info("http server shutdown complete", "component", "cmd.ws", "operation", "shutdown")
	}
}
