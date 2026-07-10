package main

import (
	"log"

	"velocity/internal/app"
)

func main() {
	container, err := app.Startup()
	if err != nil {
		log.Fatal(err)
	}

	defer app.Shutdown(container)

	container.Logger.Info("velocity started successfully")

	select {}
}