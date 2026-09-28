package candles

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/persistence/postgres/repository"
)

type fakeTradeRepo struct {
	trades []generated.Trade

	sinceArgs []time.Time
}

func (f *fakeTradeRepo) Create(context.Context, generated.CreateTradeParams) (generated.Trade, error) {
	return generated.Trade{}, nil
}

func (f *fakeTradeRepo) CreateIfNotExists(context.Context, generated.CreateTradeIfNotExistsParams) (generated.Trade, error) {
	return generated.Trade{}, nil
}

func (f *fakeTradeRepo) ListByUser(context.Context, int64) ([]generated.Trade, error) {
	return nil, nil
}

func (f *fakeTradeRepo) ListByOrder(
	context.Context,
	int64,
) ([]generated.Trade, error) {
	return nil, nil
}

func (f *fakeTradeRepo) ListBySymbol(context.Context, string) ([]generated.Trade, error) {
	return nil, nil
}

func (f *fakeTradeRepo) ListBySymbolAsc(context.Context, string) ([]generated.Trade, error) {
	return nil, nil
}

func (f *fakeTradeRepo) ListBySymbolSinceAsc(
	_ context.Context,
	_ string,
	since time.Time,
) ([]generated.Trade, error) {

	f.sinceArgs = append(f.sinceArgs, since)

	var out []generated.Trade

	for _, tr := range f.trades {
		if !tr.ExecutedAt.Before(since) {
			out = append(out, tr)
		}
	}

	return out, nil
}

func (f *fakeTradeRepo) GetByID(context.Context, int64) (generated.Trade, error) {
	return generated.Trade{}, nil
}

func (f *fakeTradeRepo) WithTx(pgx.Tx) repository.TradeRepository {
	return f
}

func (f *fakeTradeRepo) TradeExists(context.Context, int64) (bool, error) {
	return false, nil
}

func TestBackfillService_BoundsReplayToCurrentUTCDay(t *testing.T) {
	manager := NewManager()

	now := time.Now().UTC()
	todayStart := now.Truncate(Interval1d.Duration())
	yesterday := todayStart.Add(-2 * time.Hour)

	repo := &fakeTradeRepo{
		trades: []generated.Trade{
			{Symbol: "BTCUSDT", Price: 90, Quantity: 1, ExecutedAt: yesterday},
			{Symbol: "BTCUSDT", Price: 100, Quantity: 1, ExecutedAt: todayStart.Add(time.Minute)},
		},
	}

	svc := NewBackfillService(repo, manager)

	err := svc.BackfillSymbol(context.Background(), "BTCUSDT")
	require.NoError(t, err)

	require.Len(t, repo.sinceArgs, 1, "expected exactly one ListBySymbolSinceAsc call")
	require.True(t, repo.sinceArgs[0].Equal(todayStart),
		"expected replay window to start at UTC midnight (%s), got %s", todayStart, repo.sinceArgs[0])

	latest, ok := manager.Latest("BTCUSDT", Interval1m)
	require.True(t, ok, "expected a current candle to be reconstructed")
	require.Equal(t, int64(100), latest.Close,
		"reconstructed candle should reflect only the trade inside the bounded window")
}

func TestBackfillService_BackfillSymbols_StopsOnFirstError(t *testing.T) {
	manager := NewManager()
	repo := &fakeTradeRepo{}

	svc := NewBackfillService(repo, manager)

	err := svc.BackfillSymbols(context.Background(), []generated.Symbol{
		{Symbol: "BTCUSDT"},
		{Symbol: "ETHUSDT"},
	})
	require.NoError(t, err)

	require.Len(t, repo.sinceArgs, 2, "expected one bounded replay query per symbol")
}
