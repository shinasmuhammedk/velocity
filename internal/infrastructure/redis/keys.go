package redis

const (
	KeyPrefix = "velocity"
)

func UserKey(userID string) string {
	return KeyPrefix + ":user:" + userID
}

func SessionKey(sessionID string) string {
	return KeyPrefix + ":session:" + sessionID
}

func RateLimitKey(identifier string) string {
	return KeyPrefix + ":rate_limit:" + identifier
}

func MarketTickerKey(symbol string) string {
	return KeyPrefix + ":market:ticker:" + symbol
}

func MarketOrderBookKey(symbol string) string {
	return KeyPrefix + ":market:orderbook:" + symbol
}
