package marketdata

import (
	"velocity/internal/domain/trade"
	"velocity/internal/engine/orderbook"
)

type Publisher struct {
	hub *Hub
}

func NewPublisher(
	hub *Hub,
) *Publisher {
	return &Publisher{
		hub: hub,
	}
}

func (p *Publisher) PublishTrade(
	t *trade.Trade,
) {

	p.hub.Broadcast(
		t.Symbol,
		Message{
			Type:   "trade",
			Symbol: t.Symbol,
			Data: TradeMessage{
				TradeID:  t.ID.String(),
				Price:    t.Price,
				Quantity: t.Quantity,
				BuyerID:  t.BuyerID,
				SellerID: t.SellerID,
			},
		},
	)
}

func (p *Publisher) PublishTicker(
	symbol string,
	book *orderbook.OrderBook,
) {

	bestBid := int64(0)
	bestAsk := int64(0)

	if bid := book.BestBid(); bid != nil {
		bestBid = bid.Price
	}

	if ask := book.BestAsk(); ask != nil {
		bestAsk = ask.Price
	}

	p.hub.Broadcast(
		symbol,
		Message{
			Type:   "ticker",
			Symbol: symbol,
			Data: TickerMessage{
				BestBid: bestBid,
				BestAsk: bestAsk,
			},
		},
	)
}


func (p *Publisher) PublishDepth(
	symbol string,
	book DepthProvider,
) {
	bids := make([]DepthLevel, 0)
	asks := make([]DepthLevel, 0)

	for _, l := range book.BidLevels(10) {
		bids = append(bids, DepthLevel{
			Price: l.Price,
			Quantity: l.Quantity,
		})
	}

	for _, l := range book.AskLevels(10) {
		asks = append(asks, DepthLevel{
			Price: l.Price,
			Quantity: l.Quantity,
		})
	}

	p.hub.Broadcast(
		symbol,
		Message{
			Type:   "depth",
			Symbol: symbol,
			Data: DepthMessage{
				Bids: bids,
				Asks: asks,
			},
		},
	)
}