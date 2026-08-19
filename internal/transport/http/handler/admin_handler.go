package handler

import (
	"github.com/gofiber/fiber/v2"

	"velocity/internal/persistence/postgres/mapper"
	"velocity/internal/service/marketservice"
	httprequest "velocity/internal/transport/http/dto/request"
	"velocity/pkg/errors"
	"velocity/pkg/response"
)

type AdminHandler struct {
	marketService *marketservice.Service
}

func NewAdminHandler(
	marketService *marketservice.Service,
) *AdminHandler {

	return &AdminHandler{
		marketService: marketService,
	}
}

// CreateSymbol registers a new tradeable symbol.
//
// Requires the caller to hold the admin role - enforced by
// middleware.RequireRole, not by this handler.
func (h *AdminHandler) CreateSymbol(c *fiber.Ctx) error {

	var req httprequest.CreateSymbolRequest

	if err := c.BodyParser(&req); err != nil {
		return response.Error(
			c,
			fiber.StatusBadRequest,
			"invalid request body",
			err.Error(),
		)
	}

	if req.Symbol == "" || req.BaseAsset == "" || req.QuoteAsset == "" {
		return response.Error(
			c,
			fiber.StatusBadRequest,
			"invalid request",
			"symbol, base_asset and quote_asset are required",
		)
	}

	symbol, err := h.marketService.CreateSymbol(
		c.Context(),
		marketservice.CreateSymbolRequest{
			Symbol:      req.Symbol,
			DisplayName: req.DisplayName,
			BaseAsset:   req.BaseAsset,
			QuoteAsset:  req.QuoteAsset,
			TickSize:    req.TickSize,
			LotSize:     req.LotSize,
			IsActive:    req.IsActive,
		},
	)

	if err != nil {

		if err == errors.ErrSymbolAlreadyExists {
			return response.Error(
				c,
				fiber.StatusConflict,
				"symbol already exists",
				err.Error(),
			)
		}

		return response.Error(
			c,
			fiber.StatusInternalServerError,
			"failed to create symbol",
			err.Error(),
		)
	}

	return response.Success(
		c,
		fiber.StatusCreated,
		"symbol created successfully",
		mapper.ToSymbolResponse(symbol),
	)
}

// UpdateSymbolStatus enables or disables trading for an existing symbol.
//
// Requires the caller to hold the admin role - enforced by
// middleware.RequireRole, not by this handler.
func (h *AdminHandler) UpdateSymbolStatus(c *fiber.Ctx) error {

	symbol := c.Params("symbol")

	if symbol == "" {
		return response.Error(
			c,
			fiber.StatusBadRequest,
			"invalid request",
			"symbol is required",
		)
	}

	var req httprequest.UpdateSymbolStatusRequest

	if err := c.BodyParser(&req); err != nil {
		return response.Error(
			c,
			fiber.StatusBadRequest,
			"invalid request body",
			err.Error(),
		)
	}

	if err := h.marketService.UpdateSymbolStatus(
		c.Context(),
		symbol,
		req.IsActive,
	); err != nil {

		if err == errors.ErrSymbolNotFound {
			return response.Error(
				c,
				fiber.StatusNotFound,
				"symbol not found",
				err.Error(),
			)
		}

		return response.Error(
			c,
			fiber.StatusInternalServerError,
			"failed to update symbol",
			err.Error(),
		)
	}

	return response.Success(
		c,
		fiber.StatusOK,
		"symbol updated successfully",
		fiber.Map{
			"symbol":    symbol,
			"is_active": req.IsActive,
		},
	)
}