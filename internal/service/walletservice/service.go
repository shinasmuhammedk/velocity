package walletservice

import (
	"context"
	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/persistence/postgres/repository"
	"velocity/pkg/errors"

	"github.com/jackc/pgx/v5/pgtype"
)

type Service struct {
	walletRepo      repository.WalletRepository
	transactionRepo repository.WalletTransactionRepository
}

func New(
	walletRepo repository.WalletRepository,
	transactionRepo repository.WalletTransactionRepository,
) *Service {
	return &Service{
		walletRepo:      walletRepo,
		transactionRepo: transactionRepo,
	}
}

func (s *Service) Get(ctx context.Context, userID int64, asset string) (generated.Wallet, error) {
	return s.walletRepo.Get(ctx, userID, asset)
}

func (s *Service) List(ctx context.Context, userID int64) ([]generated.Wallet, error) {
	return s.walletRepo.List(ctx, userID)
}

func (s *Service) Create(ctx context.Context, params generated.CreateWalletParams) (generated.Wallet, error) {
	return s.walletRepo.Create(ctx, params)
}

func (s *Service) Update(ctx context.Context, params generated.UpdateWalletParams) error {
	return s.walletRepo.Update(ctx, params)
}

func (s *Service) LockFunds(
	ctx context.Context,
	userID int64,
	asset string,
	amount int64,
) error {

	if amount <= 0 {
		return errors.ErrInvalidQuantity
	}

	wallet, err := s.walletRepo.Get(ctx, userID, asset)
	if err != nil {
		return err
	}

	return s.walletRepo.LockFunds(
		ctx,
		wallet.ID,
		amount,
	)
}

func (s *Service) UnlockFunds(
	ctx context.Context,
	userID int64,
	asset string,
	amount int64,
) error {

	if amount <= 0 {
		return errors.ErrInvalidQuantity
	}

	// A plain (non-locking) Get is still needed here purely to resolve
	// the wallet's UUID from (userID, asset) — the actual balance
	// mutation below is a single atomic, guarded UPDATE, so nothing
	// about correctness depends on this read being fresh or exclusive.
	wallet, err := s.walletRepo.Get(
		ctx,
		userID,
		asset,
	)
	if err != nil {
		return err
	}

	return s.walletRepo.UnlockFunds(
		ctx,
		wallet.ID,
		amount,
	)
}

func (s *Service) Deposit(
	ctx context.Context,
	userID int64,
	asset string,
	amount int64,
) error {
	if amount <= 0 {
		return errors.ErrInvalidQuantity
	}

	wallet, err := s.walletRepo.GetForUpdate(
		ctx,
		userID,
		asset,
	)
	if err != nil {
		return err
	}

	_, err = s.transactionRepo.Create(
		ctx,
		generated.CreateWalletTransactionParams{
			UserID: userID,
			Asset:  asset,
			Amount: amount,
			Type:   "DEPOSIT",
		},
	)
	if err != nil {
		return err
	}

	return s.walletRepo.Update(
		ctx,
		generated.UpdateWalletParams{
			ID:        wallet.ID,
			Available: wallet.Available + amount,
			Locked:    wallet.Locked,
		},
	)
}

func (s *Service) Withdraw(
	ctx context.Context,
	userID int64,
	asset string,
	amount int64,
) error {
	if amount <= 0 {
		return errors.ErrInvalidQuantity
	}

	wallet, err := s.walletRepo.GetForUpdate(
		ctx,
		userID,
		asset,
	)
	if err != nil {
		return err
	}

	if wallet.Available < amount {
		return errors.ErrInsufficientBalance
	}

	_, err = s.transactionRepo.Create(
		ctx,
		generated.CreateWalletTransactionParams{
			UserID: userID,
			Asset:  asset,
			Amount: amount,
			Type:   "WITHDRAWAL",
		},
	)
	if err != nil {
		return err
	}

	return s.walletRepo.Update(
		ctx,
		generated.UpdateWalletParams{
			ID:        wallet.ID,
			Available: wallet.Available - amount,
			Locked:    wallet.Locked,
		},
	)
}

func (s *Service) ConsumeLockedFunds(
	ctx context.Context,
	userID int64,
	asset string,
	amount int64,
) error {
	if amount <= 0 {
		return errors.ErrInvalidQuantity
	}

	// See UnlockFunds above: the Get here only resolves the wallet's
	// UUID. The balance change itself is one atomic, guarded UPDATE.
	wallet, err := s.walletRepo.Get(
		ctx,
		userID,
		asset,
	)
	if err != nil {
		return err
	}

	return s.walletRepo.ConsumeLockedFunds(
		ctx,
		wallet.ID,
		amount,
	)
}

func (s *Service) GetWalletByAsset(
	ctx context.Context,
	userID int64,
	asset string,
) (generated.Wallet, error) {

	return s.walletRepo.Get(
		ctx,
		userID,
		asset,
	)
}

func (s *Service) ListTransactions(
	ctx context.Context,
	userID int64,
) ([]generated.WalletTransaction, error) {
	return s.transactionRepo.ListByUser(ctx, userID)
}

func (s *Service) ListTransactionsByAsset(
	ctx context.Context,
	userID int64,
	asset string,
) ([]generated.WalletTransaction, error) {
	return s.transactionRepo.ListByUserAndAsset(
		ctx,
		generated.ListWalletTransactionsByUserAndAssetParams{
			UserID: userID,
			Asset:  asset,
		},
	)
}

func (s *Service) CreditFromTrade(
	ctx context.Context,
	userID int64,
	asset string,
	amount int64,
	tradeID int64,
) error {
	if amount <= 0 {
		return errors.ErrInvalidQuantity
	}

	wallet, err := s.walletRepo.GetForUpdate(ctx, userID, asset)
	if err != nil {
		return err
	}

	_, err = s.transactionRepo.Create(
		ctx,
		generated.CreateWalletTransactionParams{
			UserID:  userID,
			Asset:   asset,
			Amount:  amount,
			Type:    "TRADE_CREDIT",
			TradeID: pgtype.Int8{Int64: tradeID, Valid: true},
		},
	)
	if err != nil {
		return err
	}

	return s.walletRepo.Update(
		ctx,
		generated.UpdateWalletParams{
			ID:        wallet.ID,
			Available: wallet.Available + amount,
			Locked:    wallet.Locked,
		},
	)
}

func (s *Service) DebitFromTrade(
	ctx context.Context,
	userID int64,
	asset string,
	amount int64,
	tradeID int64,
) error {
	if amount <= 0 {
		return errors.ErrInvalidQuantity
	}

	wallet, err := s.walletRepo.GetForUpdate(ctx, userID, asset)
	if err != nil {
		return err
	}

	if wallet.Available < amount {
		return errors.ErrInsufficientBalance
	}

	_, err = s.transactionRepo.Create(
		ctx,
		generated.CreateWalletTransactionParams{
			UserID:  userID,
			Asset:   asset,
			Amount:  amount,
			Type:    "TRADE_DEBIT",
			TradeID: pgtype.Int8{Int64: tradeID, Valid: true},
		},
	)
	if err != nil {
		return err
	}

	return s.walletRepo.Update(
		ctx,
		generated.UpdateWalletParams{
			ID:        wallet.ID,
			Available: wallet.Available - amount,
			Locked:    wallet.Locked,
		},
	)
}

func (s *Service) DepositFromTrade(
	ctx context.Context,
	userID int64,
	asset string,
	amount int64,
	tradeID int64,
) error {
	if amount <= 0 {
		return errors.ErrInvalidQuantity
	}

	wallet, err := s.walletRepo.GetForUpdate(
		ctx,
		userID,
		asset,
	)
	if err != nil {
		return err
	}

	_, err = s.transactionRepo.Create(
		ctx,
		generated.CreateWalletTransactionParams{
			UserID: userID,
			Asset:  asset,
			Amount: amount,
			Type:   "TRADE_CREDIT",
			TradeID: pgtype.Int8{
				Int64: tradeID,
				Valid: true,
			},
		},
	)
	if err != nil {
		return err
	}

	return s.walletRepo.Update(
		ctx,
		generated.UpdateWalletParams{
			ID:        wallet.ID,
			Available: wallet.Available + amount,
			Locked:    wallet.Locked,
		},
	)
}

func (s *Service) ConsumeLockedFundsFromTrade(
	ctx context.Context,
	userID int64,
	asset string,
	amount int64,
	tradeID int64,
) error {
	if amount <= 0 {
		return errors.ErrInvalidQuantity
	}

	// See UnlockFunds's comment: GetForUpdate is kept here (rather than
	// a plain Get) not because the balance change needs it — that's
	// now a single atomic, guarded UPDATE — but because Settle() calls
	// this while already holding buyer and seller order row locks in
	// one transaction, and taking the wallet row lock here too keeps
	// every row this settlement touches locked for its duration,
	// which matters for the transaction's overall atomicity even
	// though it's no longer load-bearing for this specific update.
	wallet, err := s.walletRepo.GetForUpdate(
		ctx,
		userID,
		asset,
	)
	if err != nil {
		return err
	}

	_, err = s.transactionRepo.Create(
		ctx,
		generated.CreateWalletTransactionParams{
			UserID: userID,
			Asset:  asset,
			Amount: amount,
			Type:   "TRADE_DEBIT",
			TradeID: pgtype.Int8{
				Int64: tradeID,
				Valid: true,
			},
		},
	)
	if err != nil {
		return err
	}

	return s.walletRepo.ConsumeLockedFunds(
		ctx,
		wallet.ID,
		amount,
	)
}