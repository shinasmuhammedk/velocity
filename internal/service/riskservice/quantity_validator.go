package riskservice

import (
	"context"
	"velocity/pkg/errors"
)

const MaxOrderQuantity int64 = 1_000_000

type QuantityValidator struct{}

func NewQuantityValidator() *QuantityValidator {
	return &QuantityValidator{}
}

func (v *QuantityValidator) Validate(
	ctx context.Context,
	req ValidateOrderRequest,
) error {
	qty := req.Order.Quantity

	if qty <= 0 {
		return errors.ErrInvalidQuantity
	}

	if qty > MaxOrderQuantity {
		return errors.ErrQuantityTooLarge
	}

	return nil
}
