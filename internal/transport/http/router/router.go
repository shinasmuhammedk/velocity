package router

import (
	// identityclient "velocity/internal/transport/grpc/client/identity"
	"velocity/internal/transport/http/handler"
	// "velocity/internal/transport/http/middleware"

	"github.com/gofiber/fiber/v2"
)

func Register(
    app *fiber.App,
    orderHandler *handler.OrderHandler,
    marketHandler *handler.MarketDataHandler,
    walletHandler *handler.WalletHandler,
    positionHandler *handler.PositionHandler,
    auth fiber.Handler,
) {
    api := app.Group("/api")

    // Public
    RegisterMarketRoutes(api, marketHandler)

    // Protected
    private := api.Group("/", auth)

    RegisterOrderRoutes(private, orderHandler)
    RegisterWalletRoutes(private, walletHandler)
    RegisterPositionRoutes(private, positionHandler)
}