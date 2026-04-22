package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"

	"ws-service/internal/bootstrap"
)

func main() {
	app, err := bootstrap.Build()
	if err != nil {
		slog.Error("bootstrap failed", "component", "cmd.ws", "operation", "build", "error", err)
		os.Exit(1)
	}
	defer app.Close()

	go app.RunConsumer(context.Background())

	app.Logger.Info("ws started", "component", "cmd.ws", "operation", "start_server", "addr", app.HTTPServer.Addr)
	if err := app.HTTPServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		app.Logger.Error("server stopped with error", "component", "cmd.ws", "operation", "listen_and_serve", "error", err)
		os.Exit(1)
	}
}
