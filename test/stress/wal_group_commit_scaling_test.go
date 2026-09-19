package stress

import (
	"testing"
	"time"

	"velocity/internal/engine"
	"velocity/internal/engine/wal"

	"github.com/stretchr/testify/require"
)

// TestStress_WALGroupCommit_ScalesWithConcurrency measures WAL-backed
// throughput at several producer counts.
//
// This exists because group commit's benefit is bounded by how many
// commands are in flight at once, and SubmitOrder is synchronous: a
// producer blocks on its result channel until the engine has committed
// and applied its command. So N producers means at most N commands
// queued, which means a maximum batch size of N, which caps the number
// of fsyncs that can be amortized away.
//
// The consequence is that the 8-producer figure reported by
// TestStress_SustainedThroughput_WithAndWithoutRealWAL understates what
// group commit does for the API, where concurrency is set by how many
// clients are submitting at once, not by a benchmark constant.
//
// Run with:
//
//	go test ./test/stress/ -run TestStress_WALGroupCommit_ScalesWithConcurrency -v -timeout 20m
func TestStress_WALGroupCommit_ScalesWithConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("stress test")
	}

	concurrencies := []int{1, 8, 32, 128, 512}
	const duration = 5 * time.Second

	type result struct {
		producers  int
		throughput float64
		submitted  uint64
	}

	results := make([]result, 0, len(concurrencies))

	for i, producers := range concurrencies {
		dir := t.TempDir()

		walManager := wal.NewManager(dir, wal.NewJSONSerializer())

		walWriter, err := walManager.Writer("BTCUSDT")
		require.NoError(t, err)

		e := engine.New("BTCUSDT", walWriter, nil)

		throughput, submitted, failed := runThroughputPhase(
			t,
			e,
			producers,
			duration,
			int64(i+1)*100_000_000_000,
		)

		e.Stop()
		require.NoError(t, walManager.Close())

		require.Zero(t, failed, "%d producers: %d submit errors", producers, failed)
		require.NotZero(t, submitted, "%d producers: submitted zero orders", producers)

		results = append(results, result{
			producers:  producers,
			throughput: throughput,
			submitted:  submitted,
		})
	}

	t.Logf("========================================================")
	t.Logf("WAL-backed throughput vs. submit concurrency")
	t.Logf("========================================================")
	t.Logf("%10s  %18s  %12s", "producers", "orders/sec", "submitted")
	t.Logf("--------------------------------------------------------")

	for _, r := range results {
		t.Logf("%10d  %18.2f  %12d", r.producers, r.throughput, r.submitted)
	}

	t.Logf("========================================================")
	t.Logf("With group commit, throughput should climb with producer")
	t.Logf("count as larger batches amortize each fsync. Without it,")
	t.Logf("every row pays one fsync per order and the column is flat.")
	t.Logf("========================================================")
}