package handler

import (
	"github.com/gofiber/fiber/v2"

	"velocity/internal/persistence/postgres/mapper"
	"velocity/internal/service/positionservice"
	dtoresponse "velocity/internal/transport/http/dto/response"
	"velocity/internal/transport/http/middleware"
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

	userID := middleware.GetUserID(c)

	if userID == 0 {
		return response.Error(
			c,
			fiber.StatusUnauthorized,
			"invalid user",
			"user not found in authentication context",
		)
	}

	positions, err := h.service.List(
		c.Context(),
		userID,
	)

	if err != nil {
		return response.Error(
			c,
			fiber.StatusInternalServerError,
			"failed to retrieve positions",
			err.Error(),
		)
	}

	positionResponses := make([]dtoresponse.PositionResponse, len(positions))

	for i, p := range positions {
		positionResponses[i] = mapper.ToPositionResponse(p)
	}

	return response.Success(
		c,
		fiber.StatusOK,
		"positions retrieved successfully",
		positionResponses,
	)
}

func (h *PositionHandler) GetBySymbol(
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
