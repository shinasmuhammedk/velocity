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

	wallet.Get("/", walletHandler.List)

	wallet.Get("/:asset", walletHandler.GetByAsset)

	wallet.Post("/deposit", walletHandler.Deposit)

	wallet.Post("/withdraw", walletHandler.Withdraw)
}
