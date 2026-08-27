package router

import (
	"velocity/internal/transport/http/handler"

	"github.com/gofiber/fiber/v2"
)

func RegisterSellerRoutes(
	api fiber.Router,
	sellerHandler *handler.SellerHandler,
) {
	seller := api.Group("/seller")

	seller.Get("/products", sellerHandler.GetProducts)
	seller.Post("/products", sellerHandler.CreateProduct)
	
	seller.Get("/stats", sellerHandler.GetStats)
	seller.Get("/activity", sellerHandler.GetActivity)
}
