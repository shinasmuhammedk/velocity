package wal

import (
	"velocity/internal/engine"
)

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

func (r *Replayer) Replay(
	e *engine.Engine,
	fromSequence uint64,
) error {

	events, err := r.reader.ReadAll()
	if err != nil {
		return err
	}

	for _, event := range events {

		// Skip events already included in snapshot
		if event.Sequence <= fromSequence {
			continue
		}

		switch event.Type {

		case EventSubmit:
			if event.Order != nil {
				e.RecoverOrder(event.Order)
			}

		case EventCancel:
			if event.OrderID != "" {
				// Recovery-only path
				_ = e.OrderBook().CancelOrder(
					event.OrderID,
				)
			}

		case EventModify:
			if event.OrderID != "" {
				_ = e.OrderBook().ModifyOrder(
					event.OrderID,
					event.NewPrice,
					event.NewQuantity,
				)
			}
		}

		// Keep sequence in sync with replay progress
		e.SetSequence(event.Sequence)
	}

	return nil
}