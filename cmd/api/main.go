package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"velocity/internal/app"
	"velocity/pkg/logger"
)

func main() {
	container, err := app.Bootstrap()
	if err != nil {
		log.Fatal(err)
	}

	// Start gRPC Server
	go func() {
		if err := container.GRPCServer.Start(); err != nil {
			container.Logger.Error(
				"grpc server failed",
				logger.ErrorField(err),
			)
		}
	}()

	container.Logger.Info("velocity started successfully")

	// Listen for shutdown signals.
	signalCtx, stopSignal := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stopSignal()

	// Start HTTP Server.
	go func() {
		if err := container.HTTP.Listen(
			":"+strconv.Itoa(container.Config.Server.Port),
		); err != nil {
			container.Logger.Error(
				"http server failed",
				logger.ErrorField(err),
			)
		}
	}()

	// Wait for Ctrl+C / SIGTERM.
	<-signalCtx.Done()

	container.Logger.Info("shutdown signal received")

	app.Shutdown(container)
}