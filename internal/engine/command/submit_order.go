package command

import "velocity/internal/domain/order"

type SubmitOrderCommand struct {
	Order *order.Order
    Done chan struct{}
}

func (SubmitOrderCommand) isCommand(){}