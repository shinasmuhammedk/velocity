package events

import "time"

type Event interface {
	Type() EventType
	Timestamp() time.Time
}