package handler

import (
    "github.com/gofiber/fiber/v2"
    "github.com/google/uuid"

    "velocity/internal/service/positionservice"
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

    userID, err := uuid.Parse(
        c.Params("userID"),
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