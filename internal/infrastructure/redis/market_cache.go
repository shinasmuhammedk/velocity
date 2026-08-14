package redis

import (
	"context"
	"encoding/json"
	"time"
)

const (
	MarketDataCacheTTL   = 5 * time.Second
	MarketOrderBookDepth = 20
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
	)
}

func (c *MarketCache) get(
	ctx context.Context,
	key string,
	dest any,
) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}

func (c *MarketCache) set(
	ctx context.Context,
	key string,
	value any,
	ttl time.Duration,
) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.client.Set(
		ctx,
		key,
		data,
		ttl,
	).Err()
}
