package riskservice

import (
	"context"
	"velocity/pkg/constants"
	"velocity/pkg/errors"
)

type PriceValidator struct{}

func NewPriceValidator() *PriceValidator {
	return &PriceValidator{}
}

func (v *PriceValidator) Validate(
	ctx context.Context,
	req ValidateOrderRequest,
) error {
	o := req.Order

	//Market orders dont require a price
	if o.Type == constants.OrderTypeMarket {
		return nil
	}

	if o.Price <= 0 {
		return errors.ErrInvalidPrice
	}
	return nil

}
