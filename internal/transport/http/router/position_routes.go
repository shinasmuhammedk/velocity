package router

import (
    "github.com/gofiber/fiber/v2"

    "velocity/internal/transport/http/handler"
)

func RegisterPositionRoutes(
    api fiber.Router,
    positionHandler *handler.PositionHandler,
) {

    positions := api.Group("/positions")

    positions.Get(
        "/:userID",
        positionHandler.List,
    )
}