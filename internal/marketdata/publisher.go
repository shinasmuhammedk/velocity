package marketdata

import (
	"velocity/internal/domain/trade"
	"velocity/pkg/constants"
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
			Type:   constants.MessageTrade,
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
	lastPrice int64,
	book MarketDataProvider,
) {

	bestBid := book.BestBidPrice()
	bestAsk := book.BestAskPrice()

	msg := NewTickerMessage(
		symbol,
		lastPrice,
		bestBid,
		bestAsk,
	)

	p.hub.Broadcast(
		symbol,
		msg,
	)
}

func (p *Publisher) PublishDepth(
	symbol string,
	book MarketDataProvider,
) {
	bids := make([]DepthLevel, 0)
	asks := make([]DepthLevel, 0)

	for _, l := range book.BidLevels(10) {
		bids = append(bids, DepthLevel{
			Price:    l.Price,
			Quantity: l.Quantity,
		})
	}

	for _, l := range book.AskLevels(10) {
		asks = append(asks, DepthLevel{
			Price:    l.Price,
			Quantity: l.Quantity,
		})
	}

	p.hub.Broadcast(
		symbol,
		Message{
			Type:  constants.MessageDepth,
			Symbol: symbol,
			Data: DepthMessage{
				Bids: bids,
				Asks: asks,
			},
		},
	)
}
