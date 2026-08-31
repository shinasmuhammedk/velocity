package main

import (
	"context"
	"log"

	"velocity/internal/config"
	"velocity/internal/persistence/postgres"
	"velocity/seed"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := postgres.New(cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := seed.Run(ctx, db); err != nil {
		log.Fatal(err)
	}

	log.Println("Seed completed successfully")
}
