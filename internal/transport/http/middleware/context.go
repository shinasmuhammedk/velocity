package middleware

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func GetUserID(c *fiber.Ctx) int64 {
    value := c.Locals("userID")

    switch v := value.(type) {

    case int64:
        return v

    case int32:
        return int64(v)

    case int:
        return int64(v)

    case uint:
        return int64(v)

    case uint32:
        return int64(v)

    case uint64:
        return int64(v)

    case string:
        id, err := strconv.ParseInt(v, 10, 64)
        if err != nil {
            return 0
        }
        return id

    case float64:
        return int64(v)

    default:
        return 0
    }
}

func GetEmail(c *fiber.Ctx) string {
	value, _ := c.Locals("email").(string)
	return value
}

func GetRole(c *fiber.Ctx) string {
	value, _ := c.Locals("role").(string)
	return value
}