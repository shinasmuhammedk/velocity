package main

import (
	"log"

	"velocity/internal/app"
	"velocity/pkg/logger"
)

func main() {
	container, err := app.Bootstrap()
	if err != nil {
		log.Fatal(err) // fine here — nothing to shut down yet, Bootstrap failed
	}

	defer app.Shutdown(container)

	container.Logger.Info("velocity started successfully")

	if err := container.HTTP.Listen(":8080"); err != nil {
		container.Logger.Error("http server failed", logger.ErrorField(err))
		return // let the deferred Shutdown run, then exit normally
	}
}
