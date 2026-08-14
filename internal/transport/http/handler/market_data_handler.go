package handler

import (
	"context"

	"github.com/gofiber/fiber/v2"

	"velocity/internal/analytics/candles"
	"velocity/internal/service/marketservice"
	dtoresponse "velocity/internal/transport/http/dto/response"
	"velocity/internal/transport/http/middleware"
	"velocity/pkg/response"
)

type MarketDataHandler struct {
	marketService *marketservice.Service
}

func NewMarketDataHandler(
	marketService *marketservice.Service,
) *MarketDataHandler {
	return &MarketDataHandler{
		marketService: marketService,
	}
}

// GetOrderBook godoc
//
//	@Summary		Get order book
//	@Description	Returns the current order book for a symbol
//	@Tags			Market Data
//	@Produce		json
//	@Param			symbol	path		string	true	"Trading Symbol"
//	@Param			limit	query		int		false	"Depth limit"	default(20)
//	@Success		200		{object}	response.OrderBookResponse
//	@Failure		400		{object}	response.ErrorResponse
//	@Failure		404		{object}	response.ErrorResponse
//	@Router			/api/orderbook/{symbol} [get]
func (h *MarketDataHandler) GetOrderBook(c *fiber.Ctx) error {

	symbol := c.Params("symbol")
	if symbol == "" {
		return fiber.NewError(
			fiber.StatusBadRequest,
			"symbol is required",
		)
	}

	limit := c.QueryInt("limit", 20)

	orderBook, err := h.marketService.GetOrderBook(
		context.Background(),
		symbol,
		limit,
	)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(
		dtoresponse.OrderBookResponse{
			Symbol: orderBook.Symbol,
			Bids:   orderBook.Bids,
			Asks:   orderBook.Asks,
		},
	)
}

func (h *MarketDataHandler) GetTicker(c *fiber.Ctx) error {

	symbol := c.Params("symbol")

	ticker, err := h.marketService.GetTicker(context.Background(), symbol)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(ticker)
}

func (h *MarketDataHandler) GetRecentTrades(c *fiber.Ctx) error {

	symbol := c.Params("symbol")

	trades, err := h.marketService.GetRecentTrades(
		c.Context(),
		symbol,
	)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(trades)
}

func (h *MarketDataHandler) Symbols(c *fiber.Ctx) error {

	symbols, err := h.marketService.GetSymbols(
		c.Context(),
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{
				"error": err.Error(),
			})
	}

	return c.JSON(symbols)
}

func (h *MarketDataHandler) GetUserTrades(c *fiber.Ctx) error {

	userID := middleware.GetUserID(c)

	if userID == 0 {
		return response.Error(
			c,
			fiber.StatusUnauthorized,
			"invalid user",
			"user not found in authentication context",
		)
	}

	trades, err := h.marketService.GetUserTrades(
		c.Context(),
		userID,
	)

	if err != nil {
		return response.Error(
			c,
			fiber.StatusInternalServerError,
			"failed to retrieve trades",
			err.Error(),
		)
	}

	return response.Success(
		c,
		fiber.StatusOK,
		"trades retrieved successfully",
		trades,
	)
}

func (h *MarketDataHandler) GetMarketStats(c *fiber.Ctx) error {

	symbol := c.Params("symbol")

	stats, err := h.marketService.GetMarketStats(symbol)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(stats)
}

func (h *MarketDataHandler) GetCandles(
	c *fiber.Ctx,
) error {

	symbol := c.Params("symbol")

	intervalStr := c.Query("interval")
	if intervalStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{
				"error": "interval is required",
			},
		)
	}

	interval := candles.Interval(intervalStr)

	candleData, err := h.marketService.GetCandles(
		symbol,
		interval,
	)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(
			fiber.Map{
				"error": err.Error(),
			},
		)
	}

	return c.JSON(candleData)
}
