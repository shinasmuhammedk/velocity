package snapshot

import (
	"log"
	"sync/atomic"
	"time"

	"velocity/internal/infrastructure/metrics"
)

type Snapshotable interface {
	SnapshotState() *Snapshot
	Sequence() uint64
}

type Manager struct {
	writer    SnapshotWriter
	interval  time.Duration
	threshold uint64

	lastSnapshotSequence atomic.Uint64

	stop chan struct{}
	done chan struct{}
}

func NewManager(
	writer SnapshotWriter,
	interval time.Duration,
	threshold uint64,
) *Manager {
	return &Manager{
		writer:    writer,
		interval:  interval,
		threshold: threshold,
		stop:      make(chan struct{}),
		done:      make(chan struct{}),
	}
}

func (m *Manager) Start(target Snapshotable) {
	go func() {
		defer close(m.done)

		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()

		for {
			select {

			case <-ticker.C:

				currentSequence := target.Sequence()
				lastSequence := m.lastSnapshotSequence.Load()

				if currentSequence-lastSequence < m.threshold {
					continue
				}

				start := time.Now()

				snapshot := target.SnapshotState()

				if err := m.writer.Write(snapshot); err != nil {
					metrics.SnapshotFailures.Inc()

					log.Printf(
						"snapshot write failed: %v",
						err,
					)
					continue
				}

				m.lastSnapshotSequence.Store(
					currentSequence,
				)

				metrics.SnapshotsTotal.Inc()
				metrics.SnapshotDuration.Observe(
					time.Since(start).Seconds(),
				)
				// Exported as a timestamp rather than an age so that
				// staleness is computed at query time:
				// time() - velocity_snapshot_last_success_timestamp_seconds
				// keeps rising if the process wedges, whereas a
				// process-computed age would freeze with it.
				metrics.SnapshotLastSuccessTimestamp.Set(
					float64(time.Now().Unix()),
				)

			case <-m.stop:
				return
			}
		}
	}()
}

func (m *Manager) Stop() {
	close(m.stop)
	<-m.done
}