package snapshot

import (
	"fmt"
	"os"
	"path/filepath"
)

type Writer struct {
	directory  string
	serializer Serializer
}

type SnapshotWriter interface {
    Write(*Snapshot) error
}


func NewWriter(
	directory string,
	serializer Serializer,
) *Writer {
	return &Writer{
		directory:  directory,
		serializer: serializer,
	}
}

func (w *Writer) Write(
	snapshot *Snapshot,
) error {

	if err := os.MkdirAll(w.directory, 0755); err != nil {
		return err
	}

	data, err := w.serializer.Serialize(snapshot)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf(
		"%s.snapshot",
		snapshot.Symbol,
	)

	finalPath := filepath.Join(
		w.directory,
		filename,
	)

	tempPath := finalPath + ".tmp"

	if err := os.WriteFile(
		tempPath,
		data,
		0644,
	); err != nil {
		return err
	}

	if err := os.Rename(
		tempPath,
		finalPath,
	); err != nil {
		return err
	}

	return nil
}