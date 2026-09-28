package repository

import (
	"context"

	"velocity/internal/persistence/postgres/generated"

	"github.com/jackc/pgx/v5"
)

type walletTransactionRepository struct {
	q *generated.Queries
}

func NewWalletTransactionRepository(
	db generated.DBTX,
) WalletTransactionRepository {
	return &walletTransactionRepository{
		q: generated.New(db),
	}
}

func (r *walletTransactionRepository) Create(
	ctx context.Context,
	params generated.CreateWalletTransactionParams,
) (generated.WalletTransaction, error) {
	return r.q.CreateWalletTransaction(ctx, params)
}

func (r *walletTransactionRepository) ListByUser(
	ctx context.Context,
	userID int64,
) ([]generated.WalletTransaction, error) {
	return r.q.ListWalletTransactionsByUser(ctx, userID)
}

func (r *walletTransactionRepository) ListByUserAndAsset(
	ctx context.Context,
	params generated.ListWalletTransactionsByUserAndAssetParams,
) ([]generated.WalletTransaction, error) {
	return r.q.ListWalletTransactionsByUserAndAsset(ctx, params)
}

func (r *walletTransactionRepository) WithTx(
	tx pgx.Tx,
) WalletTransactionRepository {
	return &walletTransactionRepository{
		q: generated.New(tx),
	}
}
