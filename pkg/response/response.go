package response

import (
	"github.com/gofiber/fiber/v2"
)

// Response represents the standard API response format.
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

// Success sends a successful response.
func Success(c *fiber.Ctx, status int, message string, data interface{}) error {
	return c.Status(status).JSON(Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// SuccessWithMeta sends a successful response with metadata.
func SuccessWithMeta(
	c *fiber.Ctx,
	status int,
	message string,
	data interface{},
	meta interface{},
) error {
	return c.Status(status).JSON(Response{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// Error sends an error response.
func Error(
	c *fiber.Ctx,
	status int,
	message string,
	err interface{},
) error {
	return c.Status(status).JSON(Response{
		Success: false,
		Message: message,
		Error:   err,
	})
}

// BadRequest sends a 400 response.
func BadRequest(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusBadRequest, message, nil)
}

// Unauthorized sends a 401 response.
func Unauthorized(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusUnauthorized, message, nil)
}

// Forbidden sends a 403 response.
func Forbidden(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusForbidden, message, nil)
}

// NotFound sends a 404 response.
func NotFound(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusNotFound, message, nil)
}

// Conflict sends a 409 response.
func Conflict(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusConflict, message, nil)
}

// InternalServerError sends a 500 response.
func InternalServerError(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusInternalServerError, message, nil)
}