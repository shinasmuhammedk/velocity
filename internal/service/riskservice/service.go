package riskservice

import "context"

type Service struct {
	validators []Validator
}

func New(validators ...Validator) *Service {
	return &Service{
		validators: validators,
	}
}

func (s *Service) Validate(
	ctx context.Context,
	req ValidateOrderRequest,
) (*ValidationResult, error) {
	for _, validator := range s.validators {
		if err := validator.Validate(ctx, req); err != nil {

			return &ValidationResult{
				Allowed: false,
				Reason:  err.Error(),
			}, err
		}
	}

	return &ValidationResult{
		Allowed: true,
	}, nil
}
