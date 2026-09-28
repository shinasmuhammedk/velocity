package candles

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func mustParse(t *testing.T, s string) time.Time {
	t.Helper()

	tm, err := time.Parse(time.RFC3339, s)
	require.NoError(t, err)

	return tm
}

func TestManager_Latest_ReturnsCopyNotLivePointer(t *testing.T) {
	m := NewManager()

	t0 := mustParse(t, "2024-01-01T00:00:10Z")
	m.Update("BTCUSDT", 100, 1, t0)

	latest, ok := m.Latest("BTCUSDT", Interval1m)
	require.True(t, ok)
	require.Equal(t, int64(100), latest.Close)

	m.Update("BTCUSDT", 200, 1, t0.Add(5*time.Second))

	require.Equal(t, int64(100), latest.Close,
		"Latest() leaked the live pointer: snapshot changed after a later Update()")

	fresh, ok := m.Latest("BTCUSDT", Interval1m)
	require.True(t, ok)
	require.Equal(t, int64(200), fresh.Close)
}

func TestManager_ClosedChannel_EmitsOnlyOnIntervalClose(t *testing.T) {
	m := NewManager()

	t0 := mustParse(t, "2024-01-01T00:00:10Z")
	m.Update("BTCUSDT", 100, 2, t0)
	requireNoClosedCandle(t, m)

	m.Update("BTCUSDT", 110, 1, t0.Add(30*time.Second))
	requireNoClosedCandle(t, m)

	t1 := mustParse(t, "2024-01-01T00:01:05Z")
	m.Update("BTCUSDT", 120, 3, t1)

	select {
	case closed := <-m.Closed():
		require.Equal(t, Interval1m, closed.Interval)
		require.Equal(t, int64(110), closed.Close,
			"closed candle must carry the PREVIOUS bucket's last price, not the new bucket's")
		require.Equal(t, int64(3), closed.Volume, "expected 2 (first trade) + 1 (second trade)")
	default:
		t.Fatal("expected a closed 1m candle after crossing the minute boundary")
	}

	requireNoClosedCandle(t, m)
}

func requireNoClosedCandle(t *testing.T, m *Manager) {
	t.Helper()

	select {
	case c := <-m.Closed():
		t.Fatalf("did not expect a closed candle, got %+v", c)
	default:
	}
}