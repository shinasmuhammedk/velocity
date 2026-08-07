package handler

import (
	"github.com/gofiber/fiber/v2"

	"velocity/internal/persistence/postgres/mapper"
	"velocity/internal/service/walletservice"
	"velocity/internal/transport/http/dto/request"
	dtoresponse "velocity/internal/transport/http/dto/response"
	"velocity/internal/transport/http/middleware"
	"velocity/pkg/response"
)

type WalletHandler struct {
	service *walletservice.Service
}

func NewWalletHandler(
	service *walletservice.Service,
) *WalletHandler {

	return &WalletHandler{
		service: service,
	}
}

func (h *WalletHandler) List(c *fiber.Ctx) error {

	userID := middleware.GetUserID(c)

	if userID == 0 {
		return response.Error(
			c,
			fiber.StatusUnauthorized,
			"invalid user",
			"user not found in authentication context",
		)
	}

	wallets, err := h.service.List(
		c.Context(),
		userID,
	)

	if err != nil {
		return response.Error(
			c,
			fiber.StatusInternalServerError,
			"failed to retrieve wallets",
			err.Error(),
		)
	}

	walletResponses := make([]dtoresponse.WalletResponse, len(wallets))

	for i, w := range wallets {
		walletResponses[i] = mapper.ToWalletResponse(w)
	}

	return response.Success(
		c,
		fiber.StatusOK,
		"wallets retrieved successfully",
		walletResponses,
	)
}

func (h *WalletHandler) GetByAsset(
	c *fiber.Ctx,
) error {

	userID := middleware.GetUserID(c)

	if userID == 0 {
		return response.Error(
			c,
			fiber.StatusUnauthorized,
			"invalid user",
			"user not found in authentication context",
		)
	}

	asset := c.Params("asset")

	wallet, err := h.service.GetWalletByAsset(
		c.Context(),
		userID,
		asset,
	)

	if err != nil {
		return response.Error(
			c,
			fiber.StatusNotFound,
			"wallet not found",
			err.Error(),
		)
	}

	return response.Success(
		c,
		fiber.StatusOK,
		"wallet retrieved successfully",
		mapper.ToWalletResponse(wallet),
	)
}

func (h *WalletHandler) Deposit(
	c *fiber.Ctx,
) error {

	var req request.DepositWalletRequest

	if err := c.BodyParser(&req); err != nil {
		return response.Error(
			c,
			fiber.StatusBadRequest,
			"invalid request body",
			err.Error(),
		)
	}

	userID := middleware.GetUserID(c)

	if userID == 0 {
		return response.Error(
			c,
			fiber.StatusUnauthorized,
			"invalid user",
			"user not found in authentication context",
		)
	}

	err := h.service.Deposit(
		c.Context(),
		userID,
		req.Asset,
		req.Amount,
	)

	if err != nil {
		return response.Error(
			c,
			fiber.StatusBadRequest,
			"failed to deposit funds",
			err.Error(),
		)
	}

	wallet, err := h.service.GetWalletByAsset(
		c.Context(),
		userID,
		req.Asset,
	)

	if err != nil {
		return response.Error(
			c,
			fiber.StatusInternalServerError,
			"deposit succeeded but failed to retrieve wallet",
			err.Error(),
		)
	}

	return response.Success(
		c,
		fiber.StatusOK,
		"deposit completed successfully",
		mapper.ToWalletResponse(wallet),
	)
}

func (h *WalletHandler) Withdraw(
	c *fiber.Ctx,
) error {

	var req request.WithdrawWalletRequest

	if err := c.BodyParser(&req); err != nil {
		return response.Error(
			c,
			fiber.StatusBadRequest,
			"invalid request body",
			err.Error(),
		)
	}

	userID := middleware.GetUserID(c)

	if userID == 0 {
		return response.Error(
			c,
			fiber.StatusUnauthorized,
			"invalid user",
			"user not found in authentication context",
		)
	}

	err := h.service.Withdraw(
		c.Context(),
		userID,
		req.Asset,
		req.Amount,
	)

	if err != nil {
		return response.Error(
			c,
			fiber.StatusBadRequest,
			"failed to withdraw funds",
			err.Error(),
		)
	}

	wallet, err := h.service.GetWalletByAsset(
		c.Context(),
		userID,
		req.Asset,
	)

	if err != nil {
		return response.Error(
			c,
			fiber.StatusInternalServerError,
			"withdraw succeeded but failed to retrieve wallet",
			err.Error(),
		)
	}

	return response.Success(
		c,
		fiber.StatusOK,
		"withdraw completed successfully",
		mapper.ToWalletResponse(wallet),
	)
}
