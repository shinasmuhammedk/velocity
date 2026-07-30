package seed

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Run(
	ctx context.Context,
	db *pgxpool.Pool,
) error {

	if err := SeedUsers(ctx, db); err != nil {
		log.Println("SeedUsers:", err)
	}

	if err := SeedSymbols(ctx, db); err != nil {
		log.Println("SeedSymbols:", err)
	}

	if err := SeedWallets(ctx, db); err != nil {
		log.Println("SeedWallets:", err)
	}

	if err := SeedPositions(ctx, db); err != nil {
		log.Println("SeedPositions:", err)
	}

	return nil
}