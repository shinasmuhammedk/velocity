package stats

import (
	"fmt"
	"velocity/internal/engine/events"
)

type Subscriber struct {
	manager *Manager
}

func NewSubscriber(
	manager *Manager,
) *Subscriber {
	return &Subscriber{
		manager: manager,
	}
}

func (s *Subscriber) Handle(event events.Event) {
	fmt.Println(">>> CANDLE SUBSCRIBER ENTERED")

	tradeEvent, ok := event.(events.TradeExecutedEvent)
	if !ok {
		return
	}

	s.manager.Update(
		tradeEvent.Symbol,
		tradeEvent.Price,
		tradeEvent.Quantity,
		tradeEvent.Timestamp(),
	)
}
