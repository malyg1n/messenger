package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"

	"api-service/internal/bootstrap"
)

// main — точка входа api-service: собирает зависимости и запускает HTTP-сервер.
func main() {
	app, err := bootstrap.Build()
	if err != nil {
		slog.Error("bootstrap failed", "component", "cmd.api", "operation", "build", "error", err)
		os.Exit(1)
	}
	defer app.Close()

	app.Logger.Info("api started", "component", "cmd.api", "operation", "start_server", "addr", app.HTTPServer.Addr)
	if err := app.HTTPServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		app.Logger.Error("server stopped with error", "component", "cmd.api", "operation", "listen_and_serve", "error", err)
		os.Exit(1)
	}
}
