package riskservice

import (
	"context"

	"github.com/google/uuid"

	"velocity/internal/service/walletservice"
	"velocity/pkg/constants"
	"velocity/pkg/errors"
)

type BalanceValidator struct {
	walletService *walletservice.Service
}

func NewBalanceValidator(
	walletService *walletservice.Service,
) *BalanceValidator {
	return &BalanceValidator{
		walletService: walletService,
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

	userID, err := uuid.Parse(o.UserID)
	if err != nil {
		return err
	}

	wallet, err := v.walletService.Get(
		ctx,
		userID,
		"INR",
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