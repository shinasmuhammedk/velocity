package repository

import (
	"context"

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

	wallet, err := r.q.GetWallet(
		ctx,
		generated.GetWalletParams{
			UserID: userID,
			Asset:  asset,
		},
	)

	return wallet, err
}

func (r *walletRepository) Update(
	ctx context.Context,
	params generated.UpdateWalletParams,
) error {

	return r.q.UpdateWallet(ctx, params)
}

func (r *walletRepository) GetForUpdate(
	ctx context.Context,
	userID int64,
	asset string,
) (generated.Wallet, error) {

	return r.q.GetWalletForUpdate(
		ctx,
		generated.GetWalletForUpdateParams{
			UserID: userID,
			Asset:  asset,
		},
	)
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

// UnlockFunds atomically moves amount from locked back to available in a
// single conditional UPDATE, the same pattern as LockFunds above.
//
// This intentionally does NOT read the wallet first and write back a
// computed absolute value: a read-then-write-absolute-value sequence
// is only safe if every other writer of the same row participates in
// the same locking protocol, and historically not every caller did.
// A single atomic, guarded UPDATE is safe under concurrent callers
// with no such coordination required — Postgres serialises writers to
// the same row automatically.
func (r *walletRepository) UnlockFunds(
	ctx context.Context,
	walletID uuid.UUID,
	amount int64,
) error {

	rows, err := r.q.UnlockWalletFunds(
		ctx,
		generated.UnlockWalletFundsParams{
			ID:        walletID,
			Available: amount,
		},
	)

	if err != nil {
		return err
	}

	if rows == 0 {
		return errors.ErrInsufficientLockedBalance
	}

	return nil
}

// ConsumeLockedFunds atomically removes amount from locked (it has left
// the wallet entirely, e.g. paid out in a trade) in a single guarded
// UPDATE. See UnlockFunds's comment for why this is atomic rather than
// read-then-write.
func (r *walletRepository) ConsumeLockedFunds(
	ctx context.Context,
	walletID uuid.UUID,
	amount int64,
) error {

	rows, err := r.q.ConsumeWalletLockedFunds(
		ctx,
		generated.ConsumeWalletLockedFundsParams{
			ID:     walletID,
			Locked: amount,
		},
	)

	if err != nil {
		return err
	}

	if rows == 0 {
		return errors.ErrInsufficientLockedBalance
	}

	return nil
}
