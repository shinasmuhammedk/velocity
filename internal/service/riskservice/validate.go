package riskservice

import "context"

type Validator interface {
	Validate(ctx context.Context, req ValidateOrderRequest) error
}
