package marketdata

import "velocity/internal/domain/depth"

type TradeMessage struct {
	TradeID  string `json:"trade_id"`
	Price    int64  `json:"price"`
	Quantity int64  `json:"quantity"`
	BuyerID  string `json:"buyer_id"`
	SellerID string `json:"seller_id"`
}

type TickerMessage struct {
	LastPrice int64 `json:"last_price"`
	BestBid   int64 `json:"best_bid"`
	BestAsk   int64 `json:"best_ask"`
	Spread    int64 `json:"spread"`
	MidPrice  int64 `json:"mid_price"`
}

type DepthMessage struct {
	Bids []depth.Level `json:"bids"`
	Asks []depth.Level `json:"asks"`
}

type DepthProvider interface {
	BidLevels(limit int) []depth.Level
	AskLevels(limit int) []depth.Level
}