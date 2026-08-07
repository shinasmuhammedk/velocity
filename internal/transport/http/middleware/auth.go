package middleware

import (
	"fmt"
	"strings"

	"velocity/internal/transport/grpc/client/identity"

	"github.com/gofiber/fiber/v2"
)

type AuthMiddleware struct {
	identityClient *identity.Client
}

func NewAuthMiddleware(
	identityClient *identity.Client,
) *AuthMiddleware {

	return &AuthMiddleware{
		identityClient: identityClient,
	}
}

func (m *AuthMiddleware) Authenticate(c *fiber.Ctx) error {

	// -------------------------
	// Read Authorization Header
	// -------------------------

	authHeader := c.Get("Authorization")

	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "missing authorization header",
		})
	}

	// -------------------------
	// Check Bearer Token
	// -------------------------

	const bearer = "Bearer "

	if !strings.HasPrefix(authHeader, bearer) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid authorization header",
		})
	}

	token := strings.TrimPrefix(authHeader, bearer)

	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "missing token",
		})
	}

	// -------------------------
	// Validate Token
	// -------------------------

	resp, err := m.identityClient.ValidateToken(
		c.Context(),
		token,
	)

	if err != nil {

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication service unavailable",
		})
	}
	fmt.Printf("ValidateToken Response: %+v\n", resp)
	fmt.Println("Valid :", resp.Valid)
	fmt.Println("UserID:", resp.UserId)
	fmt.Println("Email :", resp.Email)
	fmt.Println("Role  :", resp.Role)

	if !resp.Valid {

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": resp.Error,
		})
	}

	// -------------------------
	// Store User Context
	// -------------------------

	c.Locals("authUser", &AuthenticatedUser{
		UserID: int64(resp.UserId),
		Email:  resp.Email,
		Role:   resp.Role,
	})

	return c.Next()
}


func (m *AuthMiddleware) AuthenticateWS(c *fiber.Ctx) error {

	token := c.Query("token")

	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "missing token",
		})
	}

	resp, err := m.identityClient.ValidateToken(
		c.Context(),
		token,
	)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication service unavailable",
		})
	}

	if !resp.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": resp.Error,
		})
	}

	c.Locals("authUser", &AuthenticatedUser{
		UserID: int64(resp.UserId),
		Email:  resp.Email,
		Role:   resp.Role,
	})

	return c.Next()
}