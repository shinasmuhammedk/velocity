package constants

type OrderSide string
type OrderType string
type TimeInForce string
type OrderStatus string
type TradeSide string

// Order Side
const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

// Order Type
const (
	OrderTypeLimit  OrderType = "LIMIT"
	OrderTypeMarket OrderType = "MARKET"
	StopMarketOrder OrderType = "STOP_MARKET"
	StopLimitOrder  OrderType = "STOP_LIMIT"
)

// Time In Force
const (
	TimeInForceGTC      TimeInForce = "GTC"
	TimeInForceIOC      TimeInForce = "IOC"
	TimeInForceFOK      TimeInForce = "FOK"
	TimeInForcePostOnly TimeInForce = "POST_ONLY"
)

// Order Status
const (
	OrderStatusPending         OrderStatus = "PENDING"
	OrderStatusOpen            OrderStatus = "OPEN"
	OrderStatusPartiallyFilled OrderStatus = "PARTIALLY_FILLED"
	OrderStatusFilled          OrderStatus = "FILLED"
	OrderStatusCancelled       OrderStatus = "CANCELLED"
	OrderStatusRejected        OrderStatus = "REJECTED"
)

// Trade Side
const (
	TradeSideBuy  TradeSide = "BUY"
	TradeSideSell TradeSide = "SELL"
)
