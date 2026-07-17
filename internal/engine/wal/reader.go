package wal

import (
	"bufio"
	"os"
)

type Reader struct {
	file       *os.File
	serializer Serializer
}

func NewReader(
	path string,
	serializer Serializer,
) (*Reader, error) {

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return &Reader{
		file:       file,
		serializer: serializer,
	}, nil
}

func (r *Reader) ReadAll() ([]*Event, error) {

	var events []*Event

	scanner := bufio.NewScanner(r.file)

	buf := make([]byte, 0, 64*1024)

	scanner.Buffer(
		buf,
		1024*1024,
	)

	for scanner.Scan() {

		event, err := r.serializer.Deserialize(
			scanner.Bytes(),
		)

		if err != nil {
			return nil, err
		}

		events = append(
			events,
			event,
		)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func (r *Reader) Close() error {
	return r.file.Close()
}