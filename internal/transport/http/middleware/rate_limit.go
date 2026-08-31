package middleware

import (
	"strconv"

	"velocity/internal/config"
	redisinfra "velocity/internal/infrastructure/redis"
	"velocity/pkg/response"

	"github.com/gofiber/fiber/v2"
)

type RateLimitMiddleware struct {
	limiter *redisinfra.RateLimiter
	config  config.RateLimitConfig
}

func NewRateLimitMiddleware(
	limiter *redisinfra.RateLimiter,
	cfg config.RateLimitConfig,
) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: limiter,
		config:  cfg,
	}
}

func (m *RateLimitMiddleware) Submit(c *fiber.Ctx) error {
	if !m.config.Enabled {
		return c.Next()
	}

	return m.limit(
		c,
		m.config.SubmitRate,
		m.config.SubmitBurst,
		"submit",
	)
}

func (m *RateLimitMiddleware) Cancel(c *fiber.Ctx) error {
	if !m.config.Enabled {
		return c.Next()
	}

	return m.limit(
		c,
		m.config.CancelRate,
		m.config.CancelBurst,
		"cancel",
	)
}

func (m *RateLimitMiddleware) Modify(c *fiber.Ctx) error {
	if !m.config.Enabled {
		return c.Next()
	}

	return m.limit(
		c,
		m.config.ModifyRate,
		m.config.ModifyBurst,
		"modify",
	)
}

func (m *RateLimitMiddleware) limit(
	c *fiber.Ctx,
	rate float64,
	burst int,
	action string,
) error {
	userID := GetUserID(c)

	if userID == 0 {
		return response.Error(
			c,
			fiber.StatusUnauthorized,
			"invalid user",
			"user not found in authentication context",
		)
	}

	var key string

	switch action {
	case "submit":
		key = redisinfra.UserSubmitRateLimitKey(
			strconv.FormatInt(userID, 10),
		)

	case "cancel":
		key = redisinfra.UserCancelRateLimitKey(
			strconv.FormatInt(userID, 10),
		)

	case "modify":
		key = redisinfra.UserModifyRateLimitKey(
			strconv.FormatInt(userID, 10),
		)

	default:
		return response.Error(
			c,
			fiber.StatusInternalServerError,
			"rate limiter configuration error",
			"unknown rate limit action",
		)
	}

	result, err := m.limiter.Allow(
		c.Context(),
		key,
		rate,
		burst,
	)

	if err != nil {
		return response.Error(
			c,
			fiber.StatusServiceUnavailable,
			"rate limiter unavailable",
			"request protection service is temporarily unavailable",
		)
	}

	c.Set(
		"X-RateLimit-Limit",
		strconv.Itoa(burst),
	)

	c.Set(
		"X-RateLimit-Remaining",
		strconv.Itoa(result.Remaining),
	)

	if !result.Allowed {
		retrySeconds := int(result.RetryAfter.Seconds())

		if retrySeconds < 1 {
			retrySeconds = 1
		}

		c.Set(
			"Retry-After",
			strconv.Itoa(retrySeconds),
		)

		return response.Error(
			c,
			fiber.StatusTooManyRequests,
			"rate limit exceeded",
			"too many requests",
		)
	}

	return c.Next()
}
