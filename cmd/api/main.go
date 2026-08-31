package main

import (
    "log"
    "strconv"

    "velocity/internal/app"
    "velocity/pkg/logger"
)

func main() {
    container, err := app.Bootstrap()
    if err != nil {
        log.Fatal(err)
    }

    defer app.Shutdown(container)

    // -----------------------------
    // Start gRPC Server
    // -----------------------------
    go func() {
        if err := container.GRPCServer.Start(); err != nil {
            container.Logger.Error(
                "grpc server failed",
                logger.ErrorField(err),
            )
        }
    }()

    defer container.GRPCServer.Stop()

    container.Logger.Info("velocity started successfully")

    // -----------------------------
    // Start HTTP Server
    // -----------------------------
    if err := container.HTTP.Listen(
        ":" + strconv.Itoa(container.Config.Server.Port),
    ); err != nil {
        container.Logger.Error(
            "http server failed",
            logger.ErrorField(err),
        )
    }
}