package handler

import (
	"strconv"
	"velocity/internal/persistence/postgres/mapper"
	"velocity/internal/service/orderservice"
	"velocity/pkg/constants"
	"velocity/pkg/response"

	httprequest "velocity/internal/transport/http/dto/request"
	httpresponse "velocity/internal/transport/http/dto/response"
	"velocity/internal/transport/http/middleware"

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

	// serviceReq := orderservice.SubmitOrderRequest{
	// 	UserID: req.UserID,
	// 	Symbol: req.Symbol,

	// 	Side:        constants.OrderSide(req.Side),
	// 	Type:        constants.OrderType(req.Type),
	// 	TimeInForce: constants.TimeInForce(req.TimeInForce),

	// 	Price:     req.Price,
	// 	StopPrice: req.StopPrice,
	// 	Quantity:  req.Quantity,
	// }

	userID := middleware.GetUserID(c)

	serviceReq := orderservice.SubmitOrderRequest{
		UserID: userID,

		Symbol: req.Symbol,

		Side: constants.OrderSide(req.Side),
		Type: constants.OrderType(req.Type),

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

func (h *OrderHandler) Cancel(c *fiber.Ctx) error {

	// userID := middleware.GetUserID(c)

	orderID, err := strconv.ParseInt(
		c.Params("id"),
		10,
		64,
	)
	err = h.orderService.Cancel(c.Context(), orderID)
	if err != nil {
		return response.Error(
			c,
			fiber.StatusBadRequest,
			"failed to cancel order",
			err.Error(),
		)
	}

	return response.Success(
		c,
		fiber.StatusOK,
		"order cancelled successfully",
		nil,
	)
}

func (h *OrderHandler) Modify(
	c *fiber.Ctx,
) error {

	orderID, err := strconv.ParseInt(
		c.Params("id"),
		10,
		64,
	)

	var req orderservice.ModifyOrderRequest

	if err := c.BodyParser(&req); err != nil {
		return response.Error(
			c,
			fiber.StatusBadRequest,
			"invalid request body",
			err.Error(),
		)
	}

	err = h.orderService.Modify(
		c.Context(),
		orderID,
		req,
	)

	if err != nil {
		return response.Error(
			c,
			fiber.StatusBadRequest,
			"failed to modify order",
			err.Error(),
		)
	}

	return response.Success(
		c,
		fiber.StatusOK,
		"order modified successfully",
		nil,
	)
}

func (h *OrderHandler) GetOpenOrders(c *fiber.Ctx) error {

	// userID := c.Params("userID")

	userID := middleware.GetUserID(c)

	orders, err := h.orderService.GetOpenOrders(
		c.Context(),
		userID,
	)

	// orders, err := h.orderService.GetOpenOrders(
	// 	c.Context(),
	// 	userID,
	// )

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(orders)
}

func (h *OrderHandler) OrderHistory(
	c *fiber.Ctx,
) error {

	// userID, err := uuid.Parse(
	// 	c.Params("userID"),
	// )

	userID := middleware.GetUserID(c)

	// if err != nil {
	// 	return c.Status(fiber.StatusBadRequest).
	// 		JSON(fiber.Map{
	// 			"error": "invalid user id",
	// 		})
	// }

	orders, err := h.orderService.ListOrderHistory(
		c.Context(),
		userID,
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{
				"error": err.Error(),
			})
	}

	return c.JSON(orders)
}

func (h *OrderHandler) GetByID(
	c *fiber.Ctx,
) error {

	orderID, err := strconv.ParseInt(
		c.Params("id"),
		10,
		64,
	)

	order, err := h.orderService.GetOrderByID(
		c.Context(),
		orderID,
	)

	if err != nil {
		return response.Error(
			c,
			fiber.StatusNotFound,
			"order not found",
			err.Error(),
		)
	}

	resp := mapper.ToOrderResponse(order)

	return response.Success(
		c,
		fiber.StatusOK,
		"order retrieved successfully",
		resp,
	)
}
