package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"velocity/internal/persistence/postgres/generated"
	"velocity/pkg/errors"
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
	userID int64,
	asset string,
) (generated.Wallet, error) {

	fmt.Println("================================")
	fmt.Println("GET WALLET")
	fmt.Println("USER :", userID)
	fmt.Println("ASSET:", asset)

	wallet, err := r.q.GetWallet(
		ctx,
		generated.GetWalletParams{
			UserID: userID,
			Asset:  asset,
		},
	)

	fmt.Println("ERR:", err)
	fmt.Println("================================")

	return wallet, err
}

func (r *walletRepository) Update(
	ctx context.Context,
	params generated.UpdateWalletParams,
) error {

	return r.q.UpdateWallet(ctx, params)
}

func (r *walletRepository) List(
	ctx context.Context,
	userID int64,
) ([]generated.Wallet, error) {

	return r.q.ListWallets(ctx, userID)
}

func (r *walletRepository) WithTx(tx pgx.Tx) WalletRepository {
	return &walletRepository{
		q: generated.New(tx),
	}
}

func (r *walletRepository) LockFunds(
	ctx context.Context,
	walletID uuid.UUID,
	amount int64,
) error {

	rows, err := r.q.LockWalletFunds(
		ctx,
		generated.LockWalletFundsParams{
			ID:        walletID,
			Available: amount,
		},
	)

	if err != nil {
		return err
	}

	if rows == 0 {
		return errors.ErrInsufficientBalance
	}

	return nil
}