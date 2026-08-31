package seed

import (
	"context"

	"velocity/internal/persistence/postgres/generated"

	"github.com/jackc/pgx/v5/pgxpool"
)

func SeedWallets(
	ctx context.Context,
	db *pgxpool.Pool,
) error {

	q := generated.New(db)

	for _, w := range Wallets {

		_, err := q.CreateWallet(
			ctx,
			generated.CreateWalletParams{
				UserID: w.UserID,

				Asset: w.Asset,

				Available: w.Available,
				Locked:    w.Locked,
			},
		)

		if err != nil {
			return err
		}
	}

	return nil
}
