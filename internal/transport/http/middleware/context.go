package middleware

import (

	"github.com/gofiber/fiber/v2"
)

type AuthenticatedUser struct {
	UserID int64
	Email  string
	Role   string
}

func GetAuthenticatedUser(c *fiber.Ctx) (*AuthenticatedUser, error) {
	user, ok := c.Locals("authUser").(*AuthenticatedUser)
	if !ok || user == nil {
		return nil, fiber.ErrUnauthorized
	}

	return user, nil
}

func GetUserID(c *fiber.Ctx) int64 {

	user, ok := c.Locals("authUser").(*AuthenticatedUser)

	if !ok || user == nil {
		return 0
	}

	return user.UserID
}

func GetEmail(c *fiber.Ctx) string {
	value, _ := c.Locals("email").(string)
	return value
}

func GetRole(c *fiber.Ctx) string {
	value, _ := c.Locals("role").(string)
	return value
}
