package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"velocity/internal/persistence/postgres/generated"
)

type walletRepository struct {
	q *generated.Queries
}

func NewWalletRepository(db generated.DBTX) WalletRepository {
	return &walletRepository{
		q: generated.New(db),
	}
}

func (r *walletRepository) Create(
	ctx context.Context,
	params generated.CreateWalletParams,
) (generated.Wallet, error) {

	return r.q.CreateWallet(ctx, params)
}

func (r *walletRepository) Get(
	ctx context.Context,
	userID uuid.UUID,
	asset string,
) (generated.Wallet, error) {

	return r.q.GetWallet(ctx, generated.GetWalletParams{
		UserID: userID,
		Asset:  asset,
	})
}

func (r *walletRepository) Update(
	ctx context.Context,
	params generated.UpdateWalletParams,
) error {

	return r.q.UpdateWallet(ctx, params)
}

func (r *walletRepository) List(
	ctx context.Context,
	userID uuid.UUID,
) ([]generated.Wallet, error) {

	return r.q.ListWallets(ctx, userID)
}

func (r *walletRepository) WithTx(tx pgx.Tx) WalletRepository {
	return &walletRepository{
		q: generated.New(tx),
	}
}