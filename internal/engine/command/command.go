package command

import "velocity/internal/domain/order"

type Kind uint8

const (
	Submit Kind = iota
	Cancel
	Modify
)

type Command struct {
	Kind        Kind
	Order       *order.Order
	OrderID     int64
	NewPrice    int64
	NewQuantity int64
	Result      chan error
	Done        chan struct{}
}