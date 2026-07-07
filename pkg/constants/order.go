package constants

// Order Side
const (
	OrderSideBuy  = "BUY"
	OrderSideSell = "SELL"
)

// Order Type
const (
	OrderTypeLimit  = "LIMIT"
	OrderTypeMarket = "MARKET"
)

// Time In Force
const (
	TimeInForceGTC = "GTC" // Good Till Cancelled
	TimeInForceIOC = "IOC" // Immediate Or Cancel
	TimeInForceFOK = "FOK" // Fill Or Kill
)

// Order Status
const (
	OrderStatusPending         = "PENDING"
	OrderStatusOpen            = "OPEN"
	OrderStatusPartiallyFilled = "PARTIALLY_FILLED"
	OrderStatusFilled          = "FILLED"
	OrderStatusCancelled       = "CANCELLED"
	OrderStatusRejected        = "REJECTED"
)

// Trade Side
const (
	TradeSideBuy  = "BUY"
	TradeSideSell = "SELL"
)