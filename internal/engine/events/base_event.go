package events

import "time"

type BaseEvent struct {
	Time time.Time
}

func NewBaseEvent() BaseEvent{
    return BaseEvent{
        Time: time.Now(),
    }
}

func (b BaseEvent) Timestamp() time.Time{
    return b.Time
}

