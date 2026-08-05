package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"velocity/internal/persistence/postgres/mapper"
	"velocity/internal/service/positionservice"
	"velocity/pkg/response"
)

type PositionHandler struct {
	service *positionservice.Service
}

func NewPositionHandler(
	service *positionservice.Service,
) *PositionHandler {

	return &PositionHandler{
		service: service,
	}
}

func (h *PositionHandler) List(
	c *fiber.Ctx,
) error {

	userID, err := strconv.ParseInt(
		c.Params("userID"),
		10,
		64,
	)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{
				"error": "invalid user id",
			},
		)
	}

	positions, err := h.service.List(
		c.Context(),
		userID,
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{
				"error": err.Error(),
			},
		)
	}

	return c.JSON(positions)
}

func (h *PositionHandler) GetBySymbol(
	c *fiber.Ctx,
) error {

	userID, err := strconv.ParseInt(
		c.Params("userID"),
		10,
		64,
	)

	if err != nil {
		return response.Error(
			c,
			fiber.StatusBadRequest,
			"invalid user id",
			err.Error(),
		)
	}

	symbol := c.Params("symbol")

	position, err := h.service.GetPosition(
		c.Context(),
		userID,
		symbol,
	)

	if err != nil {
		return response.Error(
			c,
			fiber.StatusNotFound,
			"position not found",
			err.Error(),
		)
	}

	return response.Success(
		c,
		fiber.StatusOK,
		"position retrieved successfully",
		mapper.ToPositionResponse(position),
	)
}
