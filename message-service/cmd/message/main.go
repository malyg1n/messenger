package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"message-service/internal/bootstrap"
)

// main — точка входа message-service: поднимает app и запускает Kafka consumer.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app, err := bootstrap.Build(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build app: %v\n", err)
		os.Exit(1)
	}
	defer app.Close(ctx)

	app.Logger.Info("starting message service", "component", "main", "operation", "startup")

	if err := app.Run(ctx); err != nil {
		app.Logger.Error("message service stopped with error", "component", "main", "operation", "run", "error", err)
		os.Exit(1)
	}

	app.Logger.Info("message service stopped", "component", "main", "operation", "shutdown")
}
