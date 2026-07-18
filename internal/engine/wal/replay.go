package wal

type Replayer struct {
	reader *Reader
}

func NewReplayer(
	reader *Reader,
) *Replayer {
	return &Replayer{
		reader: reader,
	}
}

// Events returns all WAL events whose sequence is greater than fromSequence.
func (r *Replayer) Events(
	fromSequence uint64,
) ([]*Event, error) {

	events, err := r.reader.ReadAll()
	if err != nil {
		return nil, err
	}

	filtered := make([]*Event, 0, len(events))

	for _, event := range events {
		if event.Sequence <= fromSequence {
			continue
		}

		filtered = append(filtered, event)
	}

	return filtered, nil
}