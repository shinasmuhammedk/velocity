package redis

import (
	"context"
	"errors"
	"testing"
	"time"
	"velocity/internal/config"

	goredis "github.com/redis/go-redis/v9"
)

type testTicker struct {
	Symbol    string `json:"symbol"`
	LastPrice int64  `json:"last_price"`
	BestBid   int64  `json:"best_bid"`
	BestAsk   int64  `json:"best_ask"`
	Spread    int64  `json:"spread"`
}

type testOrderBook struct {
	Symbol string      `json:"symbol"`
	Bids   []testLevel `json:"bids"`
	Asks   []testLevel `json:"asks"`
}

type testLevel struct {
	Price    int64 `json:"price"`
	Quantity int64 `json:"quantity"`
}

func newTestRedisClient() *Client {
	return New(configForTest())
}

func configForTest() config.RedisConfig {
	return config.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Database: 0,
	}
}

func TestMarketCache_Ticker(t *testing.T) {
	ctx := context.Background()

	client := newTestRedisClient()
	defer client.Close()

	if err := client.Ping(ctx); err != nil {
		t.Skipf("Redis is not available: %v", err)
	}

	cache := NewMarketCache(client)

	symbol := "TEST_CACHE_TICKER"

	t.Cleanup(func() {
		_ = client.Del(ctx, MarketTickerKey(symbol)).Err()
	})

	expected := testTicker{
		Symbol:    symbol,
		LastPrice: 300000,
		BestBid:   299900,
		BestAsk:   300100,
		Spread:    200,
	}

	if err := cache.SetTicker(ctx, symbol, expected); err != nil {
		t.Fatalf("SetTicker() error = %v", err)
	}

	var actual testTicker

	if err := cache.GetTicker(ctx, symbol, &actual); err != nil {
		t.Fatalf("GetTicker() error = %v", err)
	}

	if actual != expected {
		t.Fatalf("GetTicker() = %+v, want %+v", actual, expected)
	}
}

func TestMarketCache_OrderBook(t *testing.T) {
	ctx := context.Background()

	client := newTestRedisClient()
	defer client.Close()

	if err := client.Ping(ctx); err != nil {
		t.Skipf("Redis is not available: %v", err)
	}

	cache := NewMarketCache(client)

	symbol := "TEST_CACHE_ORDERBOOK"

	t.Cleanup(func() {
		_ = client.Del(ctx, MarketOrderBookKey(symbol)).Err()
	})

	expected := testOrderBook{
		Symbol: symbol,
		Bids: []testLevel{
			{Price: 299900, Quantity: 10},
			{Price: 299800, Quantity: 20},
		},
		Asks: []testLevel{
			{Price: 300100, Quantity: 15},
			{Price: 300200, Quantity: 25},
		},
	}

	if err := cache.SetOrderBook(ctx, symbol, expected); err != nil {
		t.Fatalf("SetOrderBook() error = %v", err)
	}

	var actual testOrderBook

	if err := cache.GetOrderBook(ctx, symbol, &actual); err != nil {
		t.Fatalf("GetOrderBook() error = %v", err)
	}

	if actual.Symbol != expected.Symbol {
		t.Fatalf("Symbol = %s, want %s", actual.Symbol, expected.Symbol)
	}

	if len(actual.Bids) != len(expected.Bids) {
		t.Fatalf("Bids length = %d, want %d", len(actual.Bids), len(expected.Bids))
	}

	if len(actual.Asks) != len(expected.Asks) {
		t.Fatalf("Asks length = %d, want %d", len(actual.Asks), len(expected.Asks))
	}
}

func TestMarketCache_CacheMiss(t *testing.T) {
	ctx := context.Background()

	client := newTestRedisClient()
	defer client.Close()

	if err := client.Ping(ctx); err != nil {
		t.Skipf("Redis is not available: %v", err)
	}

	cache := NewMarketCache(client)

	symbol := "TEST_CACHE_MISS"

	_ = client.Del(ctx, MarketTickerKey(symbol)).Err()

	var ticker testTicker

	err := cache.GetTicker(ctx, symbol, &ticker)

	if !errors.Is(err, goredis.Nil) {
		t.Fatalf("GetTicker() error = %v, want redis.Nil", err)
	}
}

func TestMarketCache_TTL(t *testing.T) {
	ctx := context.Background()

	client := newTestRedisClient()
	defer client.Close()

	if err := client.Ping(ctx); err != nil {
		t.Skipf("Redis is not available: %v", err)
	}

	cache := NewMarketCache(client)

	symbol := "TEST_CACHE_TTL"

	t.Cleanup(func() {
		_ = client.Del(ctx, MarketTickerKey(symbol)).Err()
	})

	value := testTicker{
		Symbol:    symbol,
		LastPrice: 300000,
	}

	if err := cache.SetTicker(ctx, symbol, value); err != nil {
		t.Fatalf("SetTicker() error = %v", err)
	}

	ttl, err := client.TTL(ctx, MarketTickerKey(symbol)).Result()
	if err != nil {
		t.Fatalf("TTL() error = %v", err)
	}

	if ttl <= 0 || ttl > MarketDataCacheTTL {
		t.Fatalf("TTL = %v, want > 0 and <= %v", ttl, MarketDataCacheTTL)
	}

	time.Sleep(100 * time.Millisecond)
}
