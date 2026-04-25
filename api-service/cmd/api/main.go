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

	"api-service/internal/bootstrap"
)

// main — точка входа api-service: собирает зависимости и запускает HTTP-сервер.
func main() {
	app, err := bootstrap.Build()
	if err != nil {
		slog.Error("bootstrap failed", "component", "cmd.api", "operation", "build", "error", err)
		os.Exit(1)
	}
	serverErrCh := make(chan error, 1)
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app.Logger.Info("api started", "component", "cmd.api", "operation", "start_server", "addr", app.HTTPServer.Addr)
	go func() {
		if err := app.HTTPServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
			return
		}
		serverErrCh <- nil
	}()

	select {
	case err := <-serverErrCh:
		if err != nil {
			app.Logger.Error("server stopped with error", "component", "cmd.api", "operation", "listen_and_serve", "error", err)
			app.Close()
			os.Exit(1)
		}

		app.Logger.Info("server stopped", "component", "cmd.api", "operation", "listen_and_serve")
		app.Close()
	case <-rootCtx.Done():
		app.Logger.Info("shutdown signal received", "component", "cmd.api", "operation", "signal", "signal", rootCtx.Err())

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := app.HTTPServer.Shutdown(shutdownCtx); err != nil {
			app.Logger.Error("graceful shutdown failed", "component", "cmd.api", "operation", "shutdown", "error", err)
			app.Close()
			os.Exit(1)
		}

		app.Logger.Info("http server shutdown complete", "component", "cmd.api", "operation", "shutdown")
		app.Close()
	}
}
