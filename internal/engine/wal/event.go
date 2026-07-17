package wal

import (
	"velocity/internal/domain/order"
	"velocity/pkg/timeutil"
)

type EventType string

const (
	EventSubmit EventType = "SUBMIT"
	EventCancel EventType = "CANCEL"
	EventModify EventType = "MODIFY"
)

type Event struct {
	Sequence  uint64    `json:"sequence"`
	Type      EventType `json:"type"`
	Symbol    string    `json:"symbol"`
	OrderID   string    `json:"order_id,omitempty"`

	Order     *order.Order `json:"order,omitempty"`

	NewPrice    int64 `json:"new_price,omitempty"`
	NewQuantity int64 `json:"new_quantity,omitempty"`

	Timestamp int64 `json:"timestamp"`
}

func NewSubmitEvent(
	sequence uint64,
	symbol string,
	order *order.Order,
) *Event {
	return &Event{
		Sequence:  sequence,
		Type:      EventSubmit,
		Symbol:    symbol,
		Order:     order,
		Timestamp: timeutil.UTCNow().UnixNano(),
	}
}

func NewCancelEvent(
	sequence uint64,
	symbol string,
	orderID string,
) *Event {
	return &Event{
		Sequence:  sequence,
		Type:      EventCancel,
		Symbol:    symbol,
		OrderID:   orderID,
		Timestamp: timeutil.UTCNow().UnixNano(),
	}
}

func NewModifyEvent(
	sequence uint64,
	symbol string,
	orderID string,
	newPrice int64,
	newQuantity int64,
) *Event {
	return &Event{
		Sequence:    sequence,
		Type:        EventModify,
		Symbol:      symbol,
		OrderID:     orderID,
		NewPrice:    newPrice,
		NewQuantity: newQuantity,
		Timestamp:   timeutil.UTCNow().UnixNano(),
	}
}