package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"velocity/internal/app"

	"go.uber.org/zap"
)

func main() {
	container, err := app.WorkerBootstrap()
	if err != nil {
		panic(err)
	}

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	container.Logger.Info("velocity worker started")

	if err := container.Consumer.Start(ctx); err != nil {
		container.Logger.Error(
			"kafka consumer stopped",
			zap.Error(err),
		)
		panic(err)
	}

	if err := container.Consumer.Close(); err != nil {
		container.Logger.Error(
			"failed to close kafka consumer",
		)
	}

	container.Logger.Info("velocity worker stopped")
}
