package riskservice

import "velocity/internal/domain/order"

type ValidateOrderRequest struct {
	Order *order.Order
}