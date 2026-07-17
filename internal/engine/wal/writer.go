package wal

import (
	"os"
	"path/filepath"
	"sync"
)

type Writer struct {
	file       *os.File
	serializer Serializer
	mu         sync.Mutex
}

func NewWriter(
	directory string,
	symbol string,
	serializer Serializer,
) (*Writer, error) {

	err := os.MkdirAll(directory, 0755)
	if err != nil {
		return nil, err
	}

	path := filepath.Join(
		directory,
		symbol+".wal",
	)

	file, err := os.OpenFile(
		path,
		os.O_CREATE|
			os.O_APPEND|
			os.O_WRONLY,
		0644,
	)
	if err != nil {
		return nil, err
	}

	return &Writer{
		file:       file,
		serializer: serializer,
	}, nil
}

func (w *Writer) Write(
	event *Event,
) error {

	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := w.serializer.Serialize(event)
	if err != nil {
		return err
	}

	data = append(data, '\n')

	_, err = w.file.Write(data)
	if err != nil {
		return err
	}

	return w.file.Sync()
}

func (w *Writer) Close() error {
	return w.file.Close()
}
