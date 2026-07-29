package candles

import (
	"fmt"

	"velocity/internal/engine/events"
)

type Subscriber struct {
	manager *Manager
}

func NewSubscriber(manager *Manager) *Subscriber {
	return &Subscriber{
		manager: manager,
	}
}

func (s *Subscriber) Handle(event events.Event) {
	fmt.Printf("EVENT TYPE: %T\n", event)

	tradeEvent, ok := event.(events.TradeExecutedEvent)
	fmt.Println("Type assertion:", ok)

	if !ok {
		return
	}

	fmt.Println("Updating candle:", tradeEvent.Symbol)

	s.manager.Update(
		tradeEvent.Symbol,
		tradeEvent.Price,
		tradeEvent.Quantity,
		tradeEvent.Timestamp(),
	)
}