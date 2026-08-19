package wal

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
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

	if event.Sequence <= w.sequence {
		return fmt.Errorf(
			"invalid WAL sequence: got %d, expected > %d",
			event.Sequence,
			w.sequence,
		)
	}

	data, err := w.serializer.Serialize(event)
	if err != nil {
		return err
	}

	data = append(data, '\n')

	if _, err := w.file.Write(data); err != nil {
		return err
	}

	if err := w.file.Sync(); err != nil {
		return err
	}

	w.sequence = event.Sequence

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
