package walletservice

import (
	"context"
	"errors"

	"velocity/internal/persistence/postgres/generated"

	"github.com/jackc/pgx/v5"
)

func (s *Service) GetOrCreateWallet(
	ctx context.Context,
	userID int64,
	asset string,
) (generated.Wallet, error) {

	wallet, err := s.walletRepo.Get(ctx, userID, asset)
	if err == nil {
		return wallet, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return generated.Wallet{}, err
	}

	return s.walletRepo.Create(
		ctx,
		generated.CreateWalletParams{
			UserID:    userID,
			Asset:     asset,
			Available: 0,
			Locked:    0,
		},
	)
}