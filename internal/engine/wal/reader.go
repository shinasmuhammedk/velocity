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

	// We look one line ahead so that a deserialize failure can be
	// classified correctly: a malformed *last* line followed by a clean
	// EOF is a torn write (the process crashed mid-Write, before the
	// record - and its trailing newline - fully reached disk), and a
	// write that never definitely completed is a write that never
	// happened. That single trailing record is discarded; everything
	// before it is the durable prefix of the log and is returned
	// normally. A malformed line anywhere else - with more lines still
	// following it - is not a torn write, it's corruption in the middle
	// of an otherwise-complete file, and that must still be a hard
	// error rather than something recovery silently papers over.
	haveLine := scanner.Scan()

	var line []byte
	if haveLine {
		line = append([]byte(nil), scanner.Bytes()...)
	}

	for haveLine {

		haveNext := scanner.Scan()

		var nextLine []byte
		if haveNext {
			nextLine = append([]byte(nil), scanner.Bytes()...)
		}

		event, err := r.serializer.Deserialize(line)

		if err != nil {
			if !haveNext && scanner.Err() == nil {
				// Torn trailing record: discard it and stop here.
				break
			}

			return nil, err
		}

		events = append(
			events,
			event,
		)

		line = nextLine
		haveLine = haveNext
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func (r *Reader) Close() error {
	return r.file.Close()
}