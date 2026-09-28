package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"velocity/internal/infrastructure/metrics"

	goredis "github.com/redis/go-redis/v9"
)

const (
	MarketDataCacheTTL   = 5 * time.Second
	MarketOrderBookDepth = 20
)

const (
	marketCacheGetTickerOperation     = "get_ticker"
	marketCacheSetTickerOperation     = "set_ticker"
	marketCacheGetOrderBookOperation  = "get_orderbook"
	marketCacheSetOrderBookOperation  = "set_orderbook"
)

type MarketCache struct {
	client *Client
}

func NewMarketCache(client *Client) *MarketCache {
	return &MarketCache{
		client: client,
	}
}

func (c *MarketCache) GetTicker(
	ctx context.Context,
	symbol string,
	dest any,
) error {
	return c.get(
		ctx,
		MarketTickerKey(symbol),
		dest,
		marketCacheGetTickerOperation,
	)
}

func (c *MarketCache) SetTicker(
	ctx context.Context,
	symbol string,
	value any,
) error {
	return c.set(
		ctx,
		MarketTickerKey(symbol),
		value,
		MarketDataCacheTTL,
		marketCacheSetTickerOperation,
	)
}

func (c *MarketCache) GetOrderBook(
	ctx context.Context,
	symbol string,
	dest any,
) error {
	return c.get(
		ctx,
		MarketOrderBookKey(symbol),
		dest,
		marketCacheGetOrderBookOperation,
	)
}

func (c *MarketCache) SetOrderBook(
	ctx context.Context,
	symbol string,
	value any,
) error {
	return c.set(
		ctx,
		MarketOrderBookKey(symbol),
		value,
		MarketDataCacheTTL,
		marketCacheSetOrderBookOperation,
	)
}

func (c *MarketCache) get(
	ctx context.Context,
	key string,
	dest any,
	operation string,
) error {
	start := time.Now()
	defer func() {
		metrics.MarketCacheOperationDuration.
			WithLabelValues(operation).
			Observe(time.Since(start).Seconds())
	}()

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			metrics.MarketCacheMisses.
				WithLabelValues(operation).
				Inc()

			return err
		}

		metrics.MarketCacheErrors.
			WithLabelValues(operation).
			Inc()

		return err
	}

	if err := json.Unmarshal(data, dest); err != nil {
		metrics.MarketCacheErrors.
			WithLabelValues(operation).
			Inc()

		return err
	}

	metrics.MarketCacheHits.
		WithLabelValues(operation).
		Inc()

	return nil
}

func (c *MarketCache) set(
	ctx context.Context,
	key string,
	value any,
	ttl time.Duration,
	operation string,
) error {
	start := time.Now()
	defer func() {
		metrics.MarketCacheOperationDuration.
			WithLabelValues(operation).
			Observe(time.Since(start).Seconds())
	}()

	data, err := json.Marshal(value)
	if err != nil {
		metrics.MarketCacheErrors.
			WithLabelValues(operation).
			Inc()

		return err
	}

	if err := c.client.Set(
		ctx,
		key,
		data,
		ttl,
	).Err(); err != nil {
		metrics.MarketCacheErrors.
			WithLabelValues(operation).
			Inc()

		return err
	}

	return nil
}