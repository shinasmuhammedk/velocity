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

func UserSubmitRateLimitKey(userID string) string {
	return RateLimitKey("submit:user:" + userID)
}

func UserCancelRateLimitKey(userID string) string {
	return RateLimitKey("cancel:user:" + userID)
}

func UserModifyRateLimitKey(userID string) string {
	return "velocity:rate_limit:modify:user:" + userID
}
