package router

import (
	"velocity/internal/transport/http/handler"

	"github.com/gofiber/fiber/v2"
)

func Register(
	app *fiber.App,
	orderHandler *handler.OrderHandler,
	marketHandler *handler.MarketDataHandler,
	walletHandler *handler.WalletHandler,
	positionHandler *handler.PositionHandler,
) {
	api := app.Group("/api")

	RegisterOrderRoutes(api, orderHandler)
	RegisterMarketRoutes(api, marketHandler)
	RegisterWalletRoutes(api, walletHandler)
	RegisterPositionRoutes(api, positionHandler)
}
