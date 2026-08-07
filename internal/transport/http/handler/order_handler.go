package handler

import (
	"strconv"
	"velocity/internal/persistence/postgres/mapper"
	"velocity/internal/service/orderservice"
	"velocity/pkg/constants"
	"velocity/pkg/response"

	httprequest "velocity/internal/transport/http/dto/request"
	httpresponse "velocity/internal/transport/http/dto/response"
    dtoresponse "velocity/internal/transport/http/dto/response"
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
		return response.Error(
			c,
			fiber.StatusBadRequest,
			"invalid request body",
			err.Error(),
		)
	}

	userID := middleware.GetUserID(c)

	if userID == 0 {
		return response.Error(
			c,
			fiber.StatusUnauthorized,
			"invalid user",
			"user not found in authentication context",
		)
	}

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
		return response.Error(
			c,
			fiber.StatusBadRequest,
			"failed to submit order",
			err.Error(),
		)
	}

	return response.Success(
		c,
		fiber.StatusCreated,
		"order submitted successfully",
		httpresponse.SubmitOrderResponse{
			OrderID: order.ID,
			Status:  string(order.Status),
			Symbol:  order.Symbol,
		},
	)
}

func (h *OrderHandler) Cancel(c *fiber.Ctx) error {

	userID := middleware.GetUserID(c)

	if userID == 0 {
		return response.Error(
			c,
			fiber.StatusUnauthorized,
			"invalid user",
			"user not found in authentication context",
		)
	}

	orderID, err := strconv.ParseInt(
		c.Params("id"),
		10,
		64,
	)

	if err != nil {
		return response.Error(
			c,
			fiber.StatusBadRequest,
			"invalid order id",
			err.Error(),
		)
	}

	err = h.orderService.Cancel(
		c.Context(),
		orderID,
		userID,
	)

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

	userID := middleware.GetUserID(c)

	if userID == 0 {
		return response.Error(
			c,
			fiber.StatusUnauthorized,
			"invalid user",
			"user not found in authentication context",
		)
	}

	orderID, err := strconv.ParseInt(
		c.Params("id"),
		10,
		64,
	)

	if err != nil {
		return response.Error(
			c,
			fiber.StatusBadRequest,
			"invalid order id",
			err.Error(),
		)
	}

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
		userID,
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

	userID := middleware.GetUserID(c)

	if userID == 0 {
		return response.Error(
			c,
			fiber.StatusUnauthorized,
			"invalid user",
			"user not found in authentication context",
		)
	}

	orders, err := h.orderService.GetOpenOrders(
		c.Context(),
		userID,
	)

	if err != nil {
		return response.Error(
			c,
			fiber.StatusBadRequest,
			"failed to retrieve open orders",
			err.Error(),
		)
	}

	orderResponses := make([]dtoresponse.OrderResponse, len(orders))

	for i, o := range orders {
		orderResponses[i] = mapper.ToOrderResponse(o)
	}

	return response.Success(
		c,
		fiber.StatusOK,
		"open orders retrieved successfully",
		orderResponses,
	)
}

func (h *OrderHandler) OrderHistory(
	c *fiber.Ctx,
) error {

	userID := middleware.GetUserID(c)

	if userID == 0 {
		return response.Error(
			c,
			fiber.StatusUnauthorized,
			"invalid user",
			"user not found in authentication context",
		)
	}

	orders, err := h.orderService.ListOrderHistory(
		c.Context(),
		userID,
	)

	if err != nil {
		return response.Error(
			c,
			fiber.StatusInternalServerError,
			"failed to retrieve order history",
			err.Error(),
		)
	}

	orderResponses := make([]dtoresponse.OrderResponse, len(orders))

	for i, o := range orders {
		orderResponses[i] = mapper.ToOrderResponse(o)
	}

	return response.Success(
		c,
		fiber.StatusOK,
		"order history retrieved successfully",
		orderResponses,
	)
}

func (h *OrderHandler) GetByID(
	c *fiber.Ctx,
) error {

	user, err := middleware.GetAuthenticatedUser(c)
	if err != nil {
		return response.Error(
			c,
			fiber.StatusUnauthorized,
			"invalid user",
			"user not found in authentication context",
		)
	}

	orderID, err := strconv.ParseInt(
		c.Params("id"),
		10,
		64,
	)

	if err != nil {
		return response.Error(
			c,
			fiber.StatusBadRequest,
			"invalid order id",
			err.Error(),
		)
	}

	order, err := h.orderService.GetUserOrderByID(
		c.Context(),
		user.UserID,
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
