package candles

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/stretchr/testify/require"

	"velocity/internal/persistence/postgres/generated"
)

type fakeCandleRepo struct {
	mu    sync.Mutex
	calls []generated.UpsertCandleParams
}

func (f *fakeCandleRepo) Upsert(
	_ context.Context,
	params generated.UpsertCandleParams,
) (generated.Candle, error) {

	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, params)

	return generated.Candle{
		Symbol:      params.Symbol,
		Interval:    params.Interval,
		OpenTime:    params.OpenTime,
		CloseTime:   params.CloseTime,
		Open:        params.Open,
		High:        params.High,
		Low:         params.Low,
		Close:       params.Close,
		Volume:      params.Volume,
		QuoteVolume: params.QuoteVolume,
		TradeCount:  params.TradeCount,
	}, nil
}

func (f *fakeCandleRepo) ListBySymbolInterval(
	context.Context,
	string,
	string,
	int32,
) ([]generated.Candle, error) {
	return nil, nil
}

func (f *fakeCandleRepo) ListBySymbolIntervalRange(
	context.Context,
	string,
	string,
	time.Time,
	time.Time,
	int32,
) ([]generated.Candle, error) {
	return nil, nil
}

func (f *fakeCandleRepo) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.calls)
}

func (f *fakeCandleRepo) lastCall() generated.UpsertCandleParams {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.calls[len(f.calls)-1]
}

func waitForCalls(t *testing.T, repo *fakeCandleRepo, n int) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)

	for time.Now().Before(deadline) {
		if repo.callCount() >= n {
			return
		}

		time.Sleep(5 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for %d Upsert call(s), got %d", n, repo.callCount())
}

func TestCandlePersister_PersistsClosedCandles(t *testing.T) {
	manager := NewManager()
	repo := &fakeCandleRepo{}

	persister := NewCandlePersister(manager, repo, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	persister.Start(ctx)

	t0 := mustParse(t, "2024-01-01T00:00:10Z")
	manager.Update("BTCUSDT", 100, 2, t0)

	t1 := mustParse(t, "2024-01-01T00:01:05Z")
	manager.Update("BTCUSDT", 120, 1, t1)

	waitForCalls(t, repo, 1)

	call := repo.lastCall()
	require.Equal(t, "BTCUSDT", call.Symbol)
	require.Equal(t, string(Interval1m), call.Interval)
	require.Equal(t, int64(100), call.Close,
		"should persist the closed candle's own close, not the new candle's")
}

func TestCandlePersister_StopsOnContextCancel(t *testing.T) {
	manager := NewManager()
	repo := &fakeCandleRepo{}

	persister := NewCandlePersister(manager, repo, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	persister.Start(ctx)
	cancel()

	time.Sleep(20 * time.Millisecond)

	manager.closed <- &Candle{Symbol: "BTCUSDT", Interval: Interval1m}

	time.Sleep(50 * time.Millisecond)
	require.Equal(t, 0, repo.callCount())
}
