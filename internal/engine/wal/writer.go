package wal

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"velocity/internal/infrastructure/metrics"
)

type Writer struct {
	file       *os.File
	serializer Serializer
	mu         sync.Mutex
	sequence   uint64

	// buf is the reusable append buffer for a batch. Holding it on the
	// writer keeps group commit allocation-free on the hot path: the
	// slice grows to the size of the largest batch seen and is then
	// reused for every subsequent one.
	buf []byte

	// single backs Write()'s one-element batch so the single-record
	// path doesn't allocate a slice per call either.
	single [1]*Event
}

func NewWriter(
	directory string,
	symbol string,
	serializer Serializer,
) (*Writer, error) {

	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, err
	}

	path := filepath.Join(
		directory,
		symbol+".wal",
	)

	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return nil, err
	}

	writer := &Writer{
		file:       file,
		serializer: serializer,
	}

	// Recover the last persisted sequence.
	reader, err := NewReader(path, serializer)
	if err != nil {
		file.Close()
		return nil, err
	}
	defer reader.Close()

	events, err := reader.ReadAll()
	if err != nil {
		file.Close()
		return nil, err
	}

	for _, event := range events {
		if event.Sequence > writer.sequence {
			writer.sequence = event.Sequence
		}
	}

	return writer, nil
}

// Write durably appends a single event.
//
// This is now a one-record batch: it has exactly the same cost and the
// same durability semantics as before (one append, one fsync), and is
// kept because recovery and the chaos/recovery tests write records one
// at a time. The engine's hot path uses WriteBatch.
func (w *Writer) Write(event *Event) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.single[0] = event

	return w.writeLocked(w.single[:])
}

// WriteBatch durably appends a batch of events with a single fsync.
//
// This is the group commit that makes WAL-backed throughput tolerable.
// fsync latency, not serialization or the append itself, dominates the
// per-record cost (see docs/performance/benchmark-results.md), so the
// whole point is that a batch of N records pays it once instead of N
// times.
//
// Durability is all-or-nothing from the caller's point of view: if this
// returns an error, the caller must treat every event in the batch as
// not written and apply none of them. Note the pre-existing caveat that
// group commit widens rather than introduces — a failed fsync means the
// records may or may not have reached disk, so recovery can legitimately
// replay events whose callers were handed an error. Closing that gap
// needs per-record checksums plus a commit marker, which is a separate
// change.
//
// Events must be in increasing sequence order and every sequence must be
// greater than the last one durably written. The whole batch is
// validated before any of it is serialized, so a bad sequence can never
// leave a partial batch on disk.
func (w *Writer) WriteBatch(events []*Event) error {
	if len(events) == 0 {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	return w.writeLocked(events)
}

// writeLocked performs the append and fsync. w.mu must be held.
func (w *Writer) writeLocked(events []*Event) error {
	// Timed from inside the lock so the histogram measures the append
	// and fsync themselves. Lock contention between concurrent writers
	// is a separate concern and would otherwise be folded into what
	// looks like disk latency.
	start := time.Now()

	// Validate the entire batch up front. Sequence gaps are fine (the
	// engine advances its counter even for commands it then fails to
	// write), but the sequence must never go backwards or repeat.
	previous := w.sequence
	for _, event := range events {
		if event.Sequence <= previous {
			metrics.WALWriteFailures.WithLabelValues("sequence").Inc()

			return fmt.Errorf(
				"invalid WAL sequence: got %d, expected > %d",
				event.Sequence,
				previous,
			)
		}

		previous = event.Sequence
	}

	// Serialize the whole batch into one buffer, so the batch reaches
	// the kernel as a single write.
	buf := w.buf[:0]

	for _, event := range events {
		data, err := w.serializer.Serialize(event)
		if err != nil {
			metrics.WALWriteFailures.WithLabelValues("serialize").Inc()
			return err
		}

		buf = append(buf, data...)
		buf = append(buf, '\n')
	}

	w.buf = buf

	if _, err := w.file.Write(buf); err != nil {
		metrics.WALWriteFailures.WithLabelValues("write").Inc()
		return err
	}

	if err := w.file.Sync(); err != nil {
		// A failed fsync means the records may or may not be durable.
		// Counting it separately from a failed write matters: this is
		// the case where recovery can legitimately disagree with what
		// the caller was told.
		metrics.WALWriteFailures.WithLabelValues("fsync").Inc()
		return err
	}

	w.sequence = events[len(events)-1].Sequence

	metrics.WALWritesTotal.Add(float64(len(events)))
	metrics.WALBytesWritten.Add(float64(len(buf)))

	// One observation per fsync, not per record: this histogram answers
	// "how long does a commit take", and dividing it by batch size would
	// hide exactly the fsync cost it exists to expose.
	metrics.WALWriteDuration.Observe(time.Since(start).Seconds())
	metrics.WALBatchSize.Observe(float64(len(events)))

	return nil
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.file.Close()
}

func (w *Writer) Sequence() uint64 {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.sequence
}