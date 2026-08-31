package middleware

import (
	"github.com/gofiber/fiber/v2"

	"velocity/pkg/constants"
)

// RequireRole returns a handler that only allows the request through
// if the authenticated user's role matches the one given.
//
// It must run after Authenticate (or AuthenticateWS), since it reads
// the AuthenticatedUser populated by those middlewares. It does not
// itself validate the token.
func RequireRole(role constants.Role) fiber.Handler {

	return func(c *fiber.Ctx) error {

		user, err := GetAuthenticatedUser(c)

		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "user not found in authentication context",
			})
		}

		if user.Role != string(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "insufficient permissions",
			})
		}

		return c.Next()
	}
}
