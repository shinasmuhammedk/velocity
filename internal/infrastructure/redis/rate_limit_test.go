package redis

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"velocity/internal/config"

	"github.com/stretchr/testify/require"
)

// newTestLimiter connects to the Redis instance configured in
// configs/config.development.yaml (same pattern used by the
// integration test suite for Postgres). Point VELOCITY_REDIS_HOST /
// VELOCITY_REDIS_PORT env vars at a different instance if needed.
func newTestLimiter(t *testing.T) (*RateLimiter, *Client) {
	t.Helper()

	cfg := config.RedisConfig{
		Host:     envOr("VELOCITY_REDIS_HOST", "localhost"),
		Port:     6379,
		Password: "",
		Database: 0,
	}

	client := New(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	require.NoError(t, client.Ping(ctx), "redis must be reachable to run rate limiter tests")

	t.Cleanup(func() {
		_ = client.Close()
	})

	return NewRateLimiter(client), client
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func uniqueKey(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("test:rate_limit:%s:%d", t.Name(), time.Now().UnixNano())
}

func TestRateLimiter_AllowsUpToBurstThenBlocks(t *testing.T) {
	limiter, client := newTestLimiter(t)
	key := uniqueKey(t)
	t.Cleanup(func() { client.Client.Del(context.Background(), key) })

	ctx := context.Background()
	rate := 1.0   // 1 token/sec refill
	burst := 5    // bucket holds 5

	// First `burst` requests should all be allowed.
	for i := 0; i < burst; i++ {
		res, err := limiter.Allow(ctx, key, rate, burst)
		require.NoError(t, err)
		require.Truef(t, res.Allowed, "request %d should be allowed (within burst)", i+1)
	}

	// The next request should be denied — bucket is empty.
	res, err := limiter.Allow(ctx, key, rate, burst)
	require.NoError(t, err)
	require.False(t, res.Allowed, "request beyond burst should be denied")
	require.Greater(t, res.RetryAfter, time.Duration(0), "RetryAfter should be positive when denied")
}

func TestRateLimiter_RefillsOverTime(t *testing.T) {
	limiter, client := newTestLimiter(t)
	key := uniqueKey(t)
	t.Cleanup(func() { client.Client.Del(context.Background(), key) })

	ctx := context.Background()
	rate := 5.0 // 5 tokens/sec -> 1 token every 200ms
	burst := 2

	// Drain the bucket.
	for i := 0; i < burst; i++ {
		res, err := limiter.Allow(ctx, key, rate, burst)
		require.NoError(t, err)
		require.True(t, res.Allowed)
	}

	res, err := limiter.Allow(ctx, key, rate, burst)
	require.NoError(t, err)
	require.False(t, res.Allowed, "bucket should be empty immediately after draining")

	// Wait long enough for at least one token to refill.
	time.Sleep(250 * time.Millisecond)

	res, err = limiter.Allow(ctx, key, rate, burst)
	require.NoError(t, err)
	require.True(t, res.Allowed, "request should be allowed after refill window")
}

func TestRateLimiter_RemainingCountDecreases(t *testing.T) {
	limiter, client := newTestLimiter(t)
	key := uniqueKey(t)
	t.Cleanup(func() { client.Client.Del(context.Background(), key) })

	ctx := context.Background()
	rate := 1.0
	burst := 10

	res1, err := limiter.Allow(ctx, key, rate, burst)
	require.NoError(t, err)
	require.True(t, res1.Allowed)

	res2, err := limiter.Allow(ctx, key, rate, burst)
	require.NoError(t, err)
	require.True(t, res2.Allowed)

	require.Lessf(
		t, res2.Remaining, res1.Remaining,
		"remaining tokens should strictly decrease on each successful call",
	)
}

func TestRateLimiter_DifferentKeysAreIndependent(t *testing.T) {
	limiter, client := newTestLimiter(t)
	keyA := uniqueKey(t) + ":a"
	keyB := uniqueKey(t) + ":b"
	t.Cleanup(func() {
		client.Client.Del(context.Background(), keyA, keyB)
	})

	ctx := context.Background()
	rate := 1.0
	burst := 1

	// Drain key A completely.
	res, err := limiter.Allow(ctx, keyA, rate, burst)
	require.NoError(t, err)
	require.True(t, res.Allowed)

	res, err = limiter.Allow(ctx, keyA, rate, burst)
	require.NoError(t, err)
	require.False(t, res.Allowed, "key A should now be exhausted")

	// Key B (e.g. a different user) must be unaffected.
	res, err = limiter.Allow(ctx, keyB, rate, burst)
	require.NoError(t, err)
	require.True(t, res.Allowed, "key B is a different bucket and should not be exhausted")
}

func TestRateLimiter_RejectsInvalidRateAndBurst(t *testing.T) {
	limiter, client := newTestLimiter(t)
	key := uniqueKey(t)
	t.Cleanup(func() { client.Client.Del(context.Background(), key) })

	ctx := context.Background()

	_, err := limiter.Allow(ctx, key, 0, 10)
	require.Error(t, err, "rate <= 0 must be rejected")

	_, err = limiter.Allow(ctx, key, 5, 0)
	require.Error(t, err, "burst <= 0 must be rejected")
}

func TestRateLimiter_KeyExpiresWhenIdle(t *testing.T) {
	limiter, client := newTestLimiter(t)
	key := uniqueKey(t)
	t.Cleanup(func() { client.Client.Del(context.Background(), key) })

	ctx := context.Background()
	rate := 10.0
	burst := 5

	_, err := limiter.Allow(ctx, key, rate, burst)
	require.NoError(t, err)

	ttl := client.Client.TTL(ctx, key).Val()
	require.Greater(t, ttl, time.Duration(0), "bucket key should have a TTL set so idle users don't leak memory in Redis")
	require.LessOrEqual(t, ttl, time.Duration(burst/int(rate)*2+1)*time.Second)
}