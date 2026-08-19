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
	healthHandler *handler.HealthHandler,
	adminHandler *handler.AdminHandler,
	auth fiber.Handler,
	requireAdmin fiber.Handler,
) {
	// Health checks live at the top level, not under /api, since
	// orchestrators (Docker, k8s) and load balancers conventionally
	// probe /health directly, and it must never require auth.
	RegisterHealthRoutes(app, healthHandler)

	api := app.Group("/api")

	// Protected
	private := api.Group("/", auth)

	private.Get("/market/trades/user", marketHandler.GetUserTrades)

	RegisterOrderRoutes(private, orderHandler)
	RegisterWalletRoutes(private, walletHandler)
	RegisterPositionRoutes(private, positionHandler)

	// Admin - auth first, then role check, in that order, so an
	// unauthenticated caller gets 401 rather than 403.
	RegisterAdminRoutes(api, adminHandler, auth, requireAdmin)

	// Public
	RegisterMarketRoutes(api, marketHandler)
}