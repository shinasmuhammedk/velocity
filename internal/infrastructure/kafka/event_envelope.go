package kafka

import "time"

type EventEnvelope struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Symbol    string    `json:"symbol"`
	Payload   any       `json:"payload"`
}
