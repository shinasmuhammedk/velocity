package main

import (
    "fmt"
    "log"
    "os"

    "github.com/golang-migrate/migrate/v4"

    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"

    _ "github.com/lib/pq"
)

func main() {
	dsn := "postgres://postgres:1234@localhost:5432/velocity_engine?sslmode=disable"

	m, err := migrate.New(
		"file://migrations",
		dsn,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer m.Close()

	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  go run ./cmd/migrate up")
		fmt.Println("  go run ./cmd/migrate down")
		fmt.Println("  go run ./cmd/migrate force <version>")
		return
	}

	switch os.Args[1] {

	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatal(err)
		}
		fmt.Println("Migrations applied successfully.")

	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatal(err)
		}
		fmt.Println("Rollback completed.")

	case "force":
		if len(os.Args) < 3 {
			log.Fatal("version required")
		}

		var version int
		fmt.Sscanf(os.Args[2], "%d", &version)

		if err := m.Force(version); err != nil {
			log.Fatal(err)
		}

		fmt.Println("Migration version forced.")

	default:
		fmt.Println("Unknown command:", os.Args[1])
	}
}