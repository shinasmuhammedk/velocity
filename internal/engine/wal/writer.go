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

func (w *Writer) Write(event *Event) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Timed from inside the lock so the histogram measures the append
	// and fsync themselves. Lock contention between concurrent writers
	// is a separate concern and would otherwise be folded into what
	// looks like disk latency.
	start := time.Now()

	if event.Sequence <= w.sequence {
		metrics.WALWriteFailures.WithLabelValues("sequence").Inc()

		return fmt.Errorf(
			"invalid WAL sequence: got %d, expected > %d",
			event.Sequence,
			w.sequence,
		)
	}

	data, err := w.serializer.Serialize(event)
	if err != nil {
		metrics.WALWriteFailures.WithLabelValues("serialize").Inc()
		return err
	}

	data = append(data, '\n')

	if _, err := w.file.Write(data); err != nil {
		metrics.WALWriteFailures.WithLabelValues("write").Inc()
		return err
	}

	if err := w.file.Sync(); err != nil {
		// A failed fsync means the record may or may not be durable.
		// Counting it separately from a failed write matters: this is
		// the case where recovery can legitimately disagree with what
		// the caller was told.
		metrics.WALWriteFailures.WithLabelValues("fsync").Inc()
		return err
	}

	w.sequence = event.Sequence

	metrics.WALWritesTotal.Inc()
	metrics.WALBytesWritten.Add(float64(len(data)))
	metrics.WALWriteDuration.Observe(time.Since(start).Seconds())

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