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
	sellerHandler *handler.SellerHandler,
	auth fiber.Handler,
) {
	api := app.Group("/api")

	// Protected
	private := api.Group("/", auth)

	private.Get("/market/trades/user", marketHandler.GetUserTrades)

	RegisterOrderRoutes(private, orderHandler)
	RegisterWalletRoutes(private, walletHandler)
	RegisterPositionRoutes(private, positionHandler)

	// Public
	RegisterMarketRoutes(api, marketHandler)

	RegisterSellerRoutes(private, sellerHandler)
}