package snapshot

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrSnapshotNotFound = errors.New(
	"snapshot not found",
)


type Loader struct {
	directory  string
	serializer Serializer
}


func NewLoader(
	directory string,
	serializer Serializer,
) *Loader {

	return &Loader{
		directory: directory,
		serializer: serializer,
	}
}


func (l *Loader) Load(
	symbol string,
) (*Snapshot, error) {


	filename := fmt.Sprintf(
		"%s.snapshot",
		symbol,
	)


	path := filepath.Join(
		l.directory,
		filename,
	)


	data, err := os.ReadFile(path)

	if err != nil {

		if os.IsNotExist(err) {
			return nil, ErrSnapshotNotFound
		}

		return nil, err
	}


	return l.serializer.Deserialize(data)
}