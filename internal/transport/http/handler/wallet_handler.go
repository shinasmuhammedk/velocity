package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"velocity/internal/persistence/postgres/mapper"
	"velocity/internal/service/walletservice"
	"velocity/internal/transport/http/dto/request"
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

	userID, err := strconv.ParseInt(
		c.Params("userID"),
		10,
		64,
	)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).
			JSON(fiber.Map{
				"error": "invalid user id",
			})
	}


	wallets, err := h.service.List(
		c.Context(),
		userID,
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{
				"error": err.Error(),
			})
	}


	return c.JSON(wallets)
}

func (h *WalletHandler) GetByAsset(
	c *fiber.Ctx,
) error {

	userID, err := strconv.ParseInt(
		c.Params("userID"),
		10,
		64,
	)

	if err != nil {
		return response.Error(
			c,
			fiber.StatusBadRequest,
			"invalid user id",
			err.Error(),
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


	userID, err := strconv.ParseInt(
		req.UserID,
		10,
		64,
	)

	if err != nil {
		return response.Error(
			c,
			fiber.StatusBadRequest,
			"invalid user id",
			err.Error(),
		)
	}


	err = h.service.Deposit(
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


	userID, err := strconv.ParseInt(
		req.UserID,
		10,
		64,
	)

	if err != nil {
		return response.Error(
			c,
			fiber.StatusBadRequest,
			"invalid user id",
			err.Error(),
		)
	}


	err = h.service.Withdraw(
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