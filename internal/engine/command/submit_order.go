package command

import "velocity/internal/domain/order"

type SubmitOrderCommand struct {
	Order *order.Order
    Result chan error
    Done chan struct{}
}

func (SubmitOrderCommand) isCommand(){}