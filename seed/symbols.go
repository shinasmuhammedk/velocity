package seed

import (
	"context"

	"velocity/internal/persistence/postgres/generated"

	"github.com/jackc/pgx/v5/pgxpool"
)

func SeedSymbols(
	ctx context.Context,
	db *pgxpool.Pool,
) error {

	q := generated.New(db)

	for _, s := range Symbols {

		_, err := q.CreateSymbol(
			ctx,
			generated.CreateSymbolParams{
				Symbol:      s.Symbol,
				DisplayName: s.DisplayName,
				BaseAsset:   s.BaseAsset,
				QuoteAsset:  s.QuoteAsset,
				TickSize:    s.TickSize,
				LotSize:     s.LotSize,
				IsActive:    s.IsActive,
			},
		)

		if err != nil {
			return err
		}
	}

	return nil
}
