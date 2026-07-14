package handler

import (
	"velocity/internal/service/orderservice"
	"velocity/pkg/constants"

	httprequest "velocity/internal/transport/http/dto/request"
	httpresponse "velocity/internal/transport/http/dto/response"

	"github.com/gofiber/fiber/v2"
)

type OrderHandler struct {
	orderService *orderservice.Service
}

func NewOrderHandler(
	orderService *orderservice.Service,
) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
	}
}

func (h *OrderHandler) Submit(c *fiber.Ctx) error {
	var req httprequest.SubmitOrderRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{
				"error": "invalid request bosy",
			},
		)
	}

	serviceReq := orderservice.SubmitOrderRequest{
		UserID: req.UserID,
		Symbol: req.Symbol,

		Side:        constants.OrderSide(req.Side),
		Type:        constants.OrderType(req.Type),
		TimeInForce: constants.TimeInForce(req.TimeInForce),

		Price:     req.Price,
		StopPrice: req.StopPrice,
		Quantity:  req.Quantity,
	}

	order, err := h.orderService.Submit(
		c.Context(),
		serviceReq,
	)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{
				"error": err.Error(),
			},
		)
	}

	return c.Status(fiber.StatusCreated).JSON(
		httpresponse.SubmitOrderResponse{
			OrderID: order.ID,
			Status:  string(order.Status),
			Symbol:  order.Symbol,
		},
	)
}
