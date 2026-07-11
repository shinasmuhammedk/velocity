package command

import "velocity/internal/domain/order"

type SubmitOrderCommand struct {
	Order *order.Order
}

func (SubmitOrderCommand) isCommand(){}