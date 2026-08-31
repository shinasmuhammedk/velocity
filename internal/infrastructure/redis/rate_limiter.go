package redis

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type RateLimitResult struct {
	Allowed    bool
	Remaining  int
	RetryAfter time.Duration
}

type RateLimiter struct {
	client *Client
}

func NewRateLimiter(
	client *Client,
) *RateLimiter {
	return &RateLimiter{
		client: client,
	}
}

var tokenBucketScript = goredis.NewScript(`
local key = KEYS[1]

local rate = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])

local time_data = redis.call("TIME")
local now_ms = tonumber(time_data[1]) * 1000 + math.floor(tonumber(time_data[2]) / 1000)

local values = redis.call("HMGET", key, "tokens", "last_refill")

local tokens = tonumber(values[1])
local last_refill = tonumber(values[2])

if tokens == nil then
    tokens = burst
    last_refill = now_ms
end

local elapsed_ms = now_ms - last_refill
local refill = (elapsed_ms / 1000.0) * rate

tokens = math.min(burst, tokens + refill)
last_refill = now_ms

local allowed = 0
local retry_after_ms = 0

if tokens >= 1 then
    tokens = tokens - 1
    allowed = 1
else
    local missing = 1 - tokens
    retry_after_ms = math.ceil((missing / rate) * 1000)
end

redis.call(
    "HSET",
    key,
    "tokens",
    tokens,
    "last_refill",
    last_refill
)

local ttl = math.ceil((burst / rate) * 2)

if ttl < 1 then
    ttl = 1
end

redis.call("EXPIRE", key, ttl)

return {
    allowed,
    math.floor(tokens),
    retry_after_ms
}
`)

func (r *RateLimiter) Allow(
	ctx context.Context,
	key string,
	rate float64,
	burst int,
) (RateLimitResult, error) {

	if rate <= 0 {
		return RateLimitResult{}, fmt.Errorf("rate limit must be greater than zero")
	}

	if burst <= 0 {
		return RateLimitResult{}, fmt.Errorf("rate limit burst must be greater than zero")
	}

	values, err := tokenBucketScript.Run(
		ctx,
		r.client.Client,
		[]string{key},
		rate,
		burst,
	).Result()

	if err != nil {
		return RateLimitResult{}, err
	}

	result, ok := values.([]interface{})
	if !ok || len(result) != 3 {
		return RateLimitResult{}, fmt.Errorf("invalid rate limiter response")
	}

	allowed, err := redisInt(result[0])
	if err != nil {
		return RateLimitResult{}, err
	}

	remaining, err := redisInt(result[1])
	if err != nil {
		return RateLimitResult{}, err
	}

	retryAfterMS, err := redisInt(result[2])
	if err != nil {
		return RateLimitResult{}, err
	}

	return RateLimitResult{
		Allowed:    allowed == 1,
		Remaining:  remaining,
		RetryAfter: time.Duration(retryAfterMS) * time.Millisecond,
	}, nil
}

func redisInt(value interface{}) (int, error) {
	switch v := value.(type) {
	case int64:
		return int(v), nil

	case string:
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0, err
		}
		return n, nil

	case []byte:
		n, err := strconv.Atoi(string(v))
		if err != nil {
			return 0, err
		}
		return n, nil

	case float64:
		return int(math.Round(v)), nil

	default:
		return 0, fmt.Errorf("unexpected redis integer type %T", value)
	}
}
