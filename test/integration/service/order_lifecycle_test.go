package integration

import (
	"fmt"
	"testing"
	"time"

	"velocity/internal/persistence/postgres/generated"
	"velocity/test/integration"

	"github.com/stretchr/testify/require"
)

func TestCreateAndGetSymbol(t *testing.T) {
	tc := integration.NewTestContext(t)

	symbolName := fmt.Sprintf("TEST_%d", time.Now().UnixNano())

	symbol, err := tc.SymbolRepo.Create(
		tc.Ctx,
		generated.CreateSymbolParams{
			Symbol:      symbolName,
			DisplayName: "Bitcoin / Tether",
			BaseAsset:   "BTC",
			QuoteAsset:  "USDT",
			TickSize:    1,
			LotSize:     1,
			IsActive:    true,
		},
	)

	require.NoError(t, err)

	require.Equal(t, symbolName, symbol.Symbol)
	require.Equal(t, "Bitcoin / Tether", symbol.DisplayName)

	dbSymbol, err := tc.SymbolRepo.GetBySymbol(
		tc.Ctx,
		symbolName,
	)

	require.NoError(t, err)

	require.Equal(t, symbol.Symbol, dbSymbol.Symbol)
	require.Equal(t, symbol.DisplayName, dbSymbol.DisplayName)
	require.Equal(t, symbol.BaseAsset, dbSymbol.BaseAsset)
	require.Equal(t, symbol.QuoteAsset, dbSymbol.QuoteAsset)
	require.Equal(t, symbol.TickSize, dbSymbol.TickSize)
	require.Equal(t, symbol.LotSize, dbSymbol.LotSize)
	require.Equal(t, symbol.IsActive, dbSymbol.IsActive)
}
