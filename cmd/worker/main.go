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

	defer container.Shutdown()

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	container.Logger.Info("velocity worker started")

	// A metrics listener that dies mid-flight must not be silent: the
	// process would keep consuming while looking unmonitored, which is
	// indistinguishable from a scrape-target outage.
	go func() {
		if err, ok := <-container.MetricsServer.Err(); ok && err != nil {
			container.Logger.Error(
				"metrics server failed",
				zap.Error(err),
			)
		}
	}()

	if err := container.Consumer.Start(ctx); err != nil {
		container.Logger.Error(
			"kafka consumer stopped",
			zap.Error(err),
		)

		if closeErr := container.Consumer.Close(); closeErr != nil {
			container.Logger.Error(
				"failed to close kafka consumer",
				zap.Error(closeErr),
			)
		}

		panic(err)
	}

	if err := container.Consumer.Close(); err != nil {
		container.Logger.Error(
			"failed to close kafka consumer",
			zap.Error(err),
		)
	}

	container.Logger.Info("velocity worker stopped")
}