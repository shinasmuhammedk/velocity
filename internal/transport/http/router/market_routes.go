package router

import (
	"github.com/gofiber/fiber/v2"

	"velocity/internal/transport/http/handler"
)

func RegisterMarketRoutes(
	api fiber.Router,
	marketHandler *handler.MarketDataHandler,
) {
	market := api.Group("/market")

	market.Get("/symbols", marketHandler.Symbols)
	market.Get("/orderbook/:symbol", marketHandler.GetOrderBook)
	market.Get("/ticker/:symbol", marketHandler.GetTicker)
	market.Get("/trades/:symbol", marketHandler.GetRecentTrades)
	market.Get("/stats/:symbol", marketHandler.GetMarketStats)
	market.Get("/:symbol/candles", marketHandler.GetCandles)
}
