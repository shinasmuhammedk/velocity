package handler

import (
	"github.com/gofiber/fiber/v2"

	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/persistence/postgres/repository"
	"velocity/internal/transport/http/middleware"
	"velocity/pkg/response"
)

type SellerHandler struct {
	repo repository.SellerRepository
}

func NewSellerHandler(repo repository.SellerRepository) *SellerHandler {
	return &SellerHandler{
		repo: repo,
	}
}

func (h *SellerHandler) GetProducts(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "invalid user", "user not found in authentication context")
	}

	products, err := h.repo.ListSellerProducts(c.Context(), userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "failed to retrieve products", err.Error())
	}

	mappedProducts := make([]fiber.Map, 0, len(products))
	for _, p := range products {
		mappedProducts = append(mappedProducts, fiber.Map{
			"id":         p.ID,
			"seller_id":  p.SellerID,
			"name":       p.Name,
			"symbol":     p.Symbol,
			"status":     p.Status,
			"created_at": p.CreatedAt,
			"updated_at": p.UpdatedAt,
			// Add dummy fields expected by the frontend to prevent crashes
			"price":  0,
			"stock":  0,
			"locked": 0,
		})
	}

	return response.Success(c, fiber.StatusOK, "products retrieved successfully", mappedProducts)
}

type CreateProductRequest struct {
	Name   string  `json:"name"`
	Symbol string  `json:"symbol"`
	Status string  `json:"status"`
	Price  float64 `json:"price"`
	Stock  int     `json:"stock"`
}

func (h *SellerHandler) CreateProduct(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "invalid user", "user not found in authentication context")
	}

	var req CreateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}

	if req.Status == "" {
		req.Status = "Active"
	}

	product, err := h.repo.CreateSellerProduct(c.Context(), generated.CreateSellerProductParams{
		SellerID: userID,
		Name:     req.Name,
		Symbol:   req.Symbol,
		Status:   req.Status,
	})
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "failed to create product", err.Error())
	}

	mappedProduct := fiber.Map{
		"id":         product.ID,
		"seller_id":  product.SellerID,
		"name":       product.Name,
		"symbol":     product.Symbol,
		"status":     product.Status,
		"created_at": product.CreatedAt,
		"updated_at": product.UpdatedAt,
		"price":      req.Price,
		"stock":      req.Stock,
		"locked":     0,
	}

	return response.Success(c, fiber.StatusCreated, "product created successfully", mappedProduct)
}

func (h *SellerHandler) GetStats(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "invalid user", "user not found in authentication context")
	}

	statsRow, err := h.repo.GetSellerStats(c.Context(), userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "failed to retrieve stats", err.Error())
	}

	stats := fiber.Map{
		"totalRevenue":         statsRow.TotalRevenue,
		"totalProductsSold":    statsRow.TotalProductsSold,
		"activeListings":       statsRow.ActiveListings,
		"lockedInventoryValue": statsRow.LockedInventoryValue,
	}
	return response.Success(c, fiber.StatusOK, "stats retrieved", stats)
}

func (h *SellerHandler) GetActivity(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "invalid user", "user not found in authentication context")
	}

	activityRows, err := h.repo.GetSellerActivity(c.Context(), userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "failed to retrieve activity", err.Error())
	}

	activity := make([]fiber.Map, 0, len(activityRows))
	for _, row := range activityRows {
		activity = append(activity, fiber.Map{
			"id":      row.ID,
			"time":    row.Time,
			"action":  row.Action,
			"product": row.Product,
			"amount":  row.Amount,
			"price":   row.Price,
		})
	}

	return response.Success(c, fiber.StatusOK, "activity retrieved", activity)
}
