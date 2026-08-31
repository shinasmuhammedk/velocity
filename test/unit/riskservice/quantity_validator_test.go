package integration

import (
	"context"
	"testing"

	"velocity/internal/domain/order"
	"velocity/internal/service/riskservice"
	"velocity/pkg/errors"

	"github.com/stretchr/testify/require"
)

func TestQuantityValidator(t *testing.T) {

	validator := riskservice.NewQuantityValidator()

	tests := []struct {
		name    string
		qty     int64
		wantErr error
	}{
		{
			name: "valid quantity",
			qty:  100,
		},
		{
			name:    "zero quantity",
			qty:     0,
			wantErr: errors.ErrInvalidQuantity,
		},
		{
			name:    "negative quantity",
			qty:     -5,
			wantErr: errors.ErrInvalidQuantity,
		},
		{
			name:    "too large",
			qty:     riskservice.MaxOrderQuantity + 1,
			wantErr: errors.ErrQuantityTooLarge,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			err := validator.Validate(
				context.Background(),
				riskservice.ValidateOrderRequest{
					Order: &order.Order{
						Quantity: tt.qty,
					},
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
