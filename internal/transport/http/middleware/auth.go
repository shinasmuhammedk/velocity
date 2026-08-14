package middleware

import (
	"strings"

	"velocity/internal/service/userservice"
	"velocity/internal/transport/grpc/client/identity"

	"github.com/gofiber/fiber/v2"
)

type AuthMiddleware struct {
	identityClient *identity.Client
	userService    *userservice.Service
}

func NewAuthMiddleware(
	identityClient *identity.Client,
	userService *userservice.Service,
) *AuthMiddleware {

	return &AuthMiddleware{
		identityClient: identityClient,
		userService:    userService,
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

	// -------------------------
	// Lazy Sync User
	// -------------------------
	_, _ = m.userService.CreateUser(c.Context(), userservice.CreateUserRequest{
		ID:    int64(resp.UserId),
		Email: resp.Email,
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

	_, _ = m.userService.CreateUser(c.Context(), userservice.CreateUserRequest{
		ID:    int64(resp.UserId),
		Email: resp.Email,
	})

	return c.Next()
}