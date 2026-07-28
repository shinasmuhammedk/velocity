package router

import (
	"github.com/gofiber/fiber/v2"

	"velocity/internal/transport/http/handler"
)

func RegisterWalletRoutes(
	api fiber.Router,
	walletHandler *handler.WalletHandler,
) {
	wallet := api.Group("/wallets")
    
    wallet.Get("/:userID", walletHandler.List)
}
