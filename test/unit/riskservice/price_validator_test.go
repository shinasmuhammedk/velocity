package integration

import (
	"context"
	"testing"

	"velocity/internal/domain/order"
	"velocity/internal/service/riskservice"
	"velocity/pkg/constants"
	"velocity/pkg/errors"

	"github.com/stretchr/testify/require"
)

func TestPriceValidator(t *testing.T) {
	validator := riskservice.NewPriceValidator()

	tests := []struct {
		name    string
		order   *order.Order
		wantErr error
	}{
		{
			name: "valid limit order",
			order: &order.Order{
				Type:  constants.OrderTypeLimit,
				Price: 50000,
			},
		},
		{
			name: "market order ignores price",
			order: &order.Order{
				Type:  constants.OrderTypeMarket,
				Price: 0,
			},
		},
		{
			name: "invalid limit price",
			order: &order.Order{
				Type:  constants.OrderTypeLimit,
				Price: 0,
			},
			wantErr: errors.ErrInvalidPrice,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			err := validator.Validate(
				context.Background(),
				riskservice.ValidateOrderRequest{
					Order: tt.order,
				},
			)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}