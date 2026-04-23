package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"message-service/internal/bootstrap"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app, err := bootstrap.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build app: %v\n", err)
		os.Exit(1)
	}
	defer app.Close()

	app.Logger.Info("starting message service", "component", "main", "operation", "startup")

	if err := app.Run(ctx); err != nil {
		app.Logger.Error("message service stopped with error", "component", "main", "operation", "run", "error", err)
		os.Exit(1)
	}

	app.Logger.Info("message service stopped", "component", "main", "operation", "shutdown")
}
