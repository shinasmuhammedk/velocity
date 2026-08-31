package integration

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewTestContext(t *testing.T) {
	tc := NewTestContext(t)

	require.NotNil(t, tc)

	require.NotNil(t, tc.Ctx)
	require.NotNil(t, tc.DB)

	require.NotNil(t, tc.UserRepo)
	require.NotNil(t, tc.OrderRepo)
	require.NotNil(t, tc.TradeRepo)
	require.NotNil(t, tc.PositionRepo)
	require.NotNil(t, tc.SymbolRepo)
	require.NotNil(t, tc.WalletRepo)
}
