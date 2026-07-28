package handler

import (
    "github.com/gofiber/fiber/v2"
    "github.com/google/uuid"

    "velocity/internal/service/walletservice"
)

type WalletHandler struct {
    service *walletservice.Service
}

func NewWalletHandler(
    service *walletservice.Service,
) *WalletHandler {

    return &WalletHandler{
        service: service,
    }
}

func (h *WalletHandler) List(c *fiber.Ctx) error {

    userID, err := uuid.Parse(
        c.Params("userID"),
    )
    if err != nil {
        return c.Status(fiber.StatusBadRequest).
            JSON(fiber.Map{
                "error": "invalid user id",
            })
    }

    wallets, err := h.service.List(
        c.Context(),
        userID,
    )
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).
            JSON(fiber.Map{
                "error": err.Error(),
            })
    }

    return c.JSON(wallets)
}