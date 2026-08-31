package middleware_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"velocity/internal/config"
	redisinfra "velocity/internal/infrastructure/redis"
	"velocity/internal/transport/http/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

// buildTestApp wires a minimal Fiber app with real auth-context
// injection (bypassing the identity gRPC call) and the real
// rate-limit middleware backed by a real Redis instance.
func buildTestApp(t *testing.T, cfg config.RateLimitConfig, userID int64) (*fiber.App, func()) {
	t.Helper()

	redisCfg := config.RedisConfig{Host: "localhost", Port: 6379}
	client := redisinfra.New(redisCfg)

	require.NoError(t, client.Ping(context.Background()))

	limiter := redisinfra.NewRateLimiter(client)
	rl := middleware.NewRateLimitMiddleware(limiter, cfg)

	app := fiber.New()

	// Fake auth: inject a user without calling the real identity service.
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("authUser", &middleware.AuthenticatedUser{UserID: userID})
		return c.Next()
	})

	app.Post("/orders", rl.Submit, func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusCreated)
	})

	cleanup := func() {
		_ = client.Client.FlushDB(context.Background()).Err()
		_ = client.Close()
	}

	return app, cleanup
}

func TestSubmitRateLimit_BlocksAfterBurst(t *testing.T) {
	cfg := config.RateLimitConfig{
		Enabled:     true,
		SubmitRate:  1,
		SubmitBurst: 3,
	}

	app, cleanup := buildTestApp(t, cfg, 9001)
	defer cleanup()

	var lastStatus int
	for i := 0; i < cfg.SubmitBurst; i++ {
		req := httptest.NewRequest("POST", "/orders", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		lastStatus = resp.StatusCode
		require.Equal(t, fiber.StatusCreated, lastStatus, "request %d within burst should succeed", i+1)
	}

	// One more should be rejected with 429 + Retry-After header.
	req := httptest.NewRequest("POST", "/orders", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusTooManyRequests, resp.StatusCode)
	require.NotEmpty(t, resp.Header.Get("Retry-After"), "429 response must include Retry-After")
	require.NotEmpty(t, resp.Header.Get("X-RateLimit-Remaining"))
}

func TestSubmitRateLimit_DisabledPassesThrough(t *testing.T) {
	cfg := config.RateLimitConfig{
		Enabled:     false, // mirrors current staging/prod config gap
		SubmitRate:  1,
		SubmitBurst: 1,
	}

	app, cleanup := buildTestApp(t, cfg, 9002)
	defer cleanup()

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("POST", "/orders", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		require.Equal(
			t, fiber.StatusCreated, resp.StatusCode,
			"with rate limiting disabled, request %d should NOT be throttled — "+
				"this passes today, which is exactly the staging/prod risk",
		)
	}
}

func TestSubmitRateLimit_UsersAreIsolated(t *testing.T) {
	cfg := config.RateLimitConfig{
		Enabled:     true,
		SubmitRate:  1,
		SubmitBurst: 1,
	}

	appA, cleanupA := buildTestApp(t, cfg, 111)
	defer cleanupA()
	appB, cleanupB := buildTestApp(t, cfg, 222)
	defer cleanupB()

	// Exhaust user A's bucket.
	req := httptest.NewRequest("POST", "/orders", nil)
	resp, err := appA.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)

	req = httptest.NewRequest("POST", "/orders", nil)
	resp, err = appA.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusTooManyRequests, resp.StatusCode)

	// User B, same endpoint, must be unaffected.
	req = httptest.NewRequest("POST", "/orders", nil)
	resp, err = appB.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, resp.StatusCode, "different user must have an independent bucket")
}