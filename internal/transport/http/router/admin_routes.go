package router

import (
	"github.com/gofiber/fiber/v2"

	"velocity/internal/transport/http/handler"
)

// RegisterAdminRoutes wires up admin-only endpoints.
//
// adminOnly must already include both the auth middleware (to identify
// the caller) and middleware.RequireRole(constants.RoleAdmin) (to
// restrict access to admins) - this function just applies whatever
// chain it's given, it doesn't build the chain itself.
func RegisterAdminRoutes(
	api fiber.Router,
	adminHandler *handler.AdminHandler,
	adminOnly ...fiber.Handler,
) {
	admin := api.Group("/admin", adminOnly...)

	admin.Post("/symbols", adminHandler.CreateSymbol)
	admin.Patch("/symbols/:symbol", adminHandler.UpdateSymbolStatus)
}