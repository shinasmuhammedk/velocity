package walletservice

import (
	"context"
	"fmt"
	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/persistence/postgres/repository"
	"velocity/pkg/errors"
)

type Service struct {
	walletRepo repository.WalletRepository
}

func New(walletRepo repository.WalletRepository) *Service {
	return &Service{
		walletRepo: walletRepo,
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

	wallet, err := s.walletRepo.Get(
		ctx,
		userID,
		asset,
	)
	if err != nil {
		return err
	}

	if wallet.Locked < amount {
		return errors.ErrInsufficientLockedBalance
	}

	return s.walletRepo.Update(
		ctx,
		generated.UpdateWalletParams{
			ID: wallet.ID,

			Available: wallet.Available + amount,
			Locked:    wallet.Locked - amount,
		},
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

	wallet, err := s.GetOrCreateWallet(
		ctx,
		userID,
		asset,
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

	wallet, err := s.walletRepo.Get(ctx, userID, asset)
	if err != nil {
		return err
	}

	if wallet.Available < amount {
		return errors.ErrInsufficientBalance
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

	wallet, err := s.walletRepo.Get(
		ctx,
		userID,
		asset,
	)
	if err != nil {
		return err
	}

	if wallet.Locked < amount {
		return errors.ErrInsufficientLockedBalance
	}

	fmt.Println("BEFORE")
	fmt.Println("Available:", wallet.Available)
	fmt.Println("Locked:", wallet.Locked)

	newLocked := wallet.Locked - amount

	fmt.Println("AFTER")
	fmt.Println("Available:", wallet.Available)
	fmt.Println("Locked:", newLocked)

	return s.walletRepo.Update(
		ctx,
		generated.UpdateWalletParams{
			ID: wallet.ID,

			Available: wallet.Available,
			Locked:    wallet.Locked - amount,
		},
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
