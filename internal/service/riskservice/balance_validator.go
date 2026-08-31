package riskservice

import (
	"context"

	"velocity/internal/persistence/postgres/repository"
	"velocity/internal/service/walletservice"
	"velocity/pkg/constants"
	"velocity/pkg/errors"
)

type BalanceValidator struct {
	walletService *walletservice.Service
	symbolRepo    repository.SymbolRepository
}

func NewBalanceValidator(
	walletService *walletservice.Service,
	symbolRepo repository.SymbolRepository,
) *BalanceValidator {
	return &BalanceValidator{
		walletService: walletService,
		symbolRepo:    symbolRepo,
	}
}

func (v *BalanceValidator) Validate(
	ctx context.Context,
	req ValidateOrderRequest,
) error {

	o := req.Order

	// Balance validation is only required for BUY orders.
	if o.Side != constants.OrderSideBuy {
		return nil
	}

	userID := o.UserID

	symbol, err := v.symbolRepo.Get(ctx, o.Symbol)
	if err != nil {
		return err
	}

	wallet, err := v.walletService.Get(
		ctx,
		userID,
		symbol.QuoteAsset,
	)
	if err != nil {
		return err
	}

	required := o.Price * o.Quantity

	if wallet.Available < required {
		return errors.ErrInsufficientBalance
	}

	return nil
}
