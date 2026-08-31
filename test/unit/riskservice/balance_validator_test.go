package integration

import (
	"context"
	"testing"

	"velocity/internal/domain/order"
	"velocity/internal/service/riskservice"

	"github.com/stretchr/testify/require"
)

type mockValidator struct {
	err error
}

func (m mockValidator) Validate(
	ctx context.Context,
	req riskservice.ValidateOrderRequest,
) error {
	return m.err
}

func TestRiskServiceSuccess(t *testing.T) {

	service := riskservice.New(
		mockValidator{},
		mockValidator{},
	)

	result, err := service.Validate(
		context.Background(),
		riskservice.ValidateOrderRequest{
			Order: &order.Order{},
		},
	)

	require.NoError(t, err)
	require.True(t, result.Allowed)
}

func TestRiskServiceFailure(t *testing.T) {

	expected := assertErr{}

	service := riskservice.New(
		mockValidator{},
		mockValidator{err: expected},
	)

	result, err := service.Validate(
		context.Background(),
		riskservice.ValidateOrderRequest{
			Order: &order.Order{},
		},
	)

	require.Error(t, err)
	require.False(t, result.Allowed)
	require.Equal(t, expected.Error(), result.Reason)
}

type assertErr struct{}

func (assertErr) Error() string {
	return "validation failed"
}
