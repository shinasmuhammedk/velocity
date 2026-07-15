package marketdata


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

type DepthLevel struct {
	Price    int64 `json:"price"`
	Quantity int64 `json:"quantity"`
}

type DepthMessage struct {
	Bids []DepthLevel `json:"bids"`
	Asks []DepthLevel `json:"asks"`
}
