package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"velocity/internal/service/marketdataservice"
	"velocity/internal/transport/http/dto/response"
)

type MarketDataHandler struct {
	marketDataService *marketdataservice.Service
}

func NewMarketDataHandler(
	marketDataService *marketdataservice.Service,
) *MarketDataHandler {
	return &MarketDataHandler{
		marketDataService: marketDataService,
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

	orderBook, err := h.marketDataService.GetOrderBook(
		symbol,
		limit,
	)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(
		response.OrderBookResponse{
			Symbol: orderBook.Symbol,
			Bids:   orderBook.Bids,
			Asks:   orderBook.Asks,
		},
	)
}

func (h *MarketDataHandler) GetTicker(c *fiber.Ctx) error {

	symbol := c.Params("symbol")

	ticker, err := h.marketDataService.GetTicker(symbol)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(ticker)
}

func (h *MarketDataHandler) GetRecentTrades(c *fiber.Ctx) error {

	symbol := c.Params("symbol")

	trades, err := h.marketDataService.GetRecentTrades(
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

    symbols, err := h.marketDataService.GetSymbols(
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

	userID, err := uuid.Parse(
		c.Params("userID"),
	)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).
			JSON(fiber.Map{
				"error": "invalid user id",
			})
	}

	trades, err := h.marketDataService.GetUserTrades(
		c.Context(),
		userID,
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{
				"error": err.Error(),
			})
	}

	return c.JSON(trades)
}