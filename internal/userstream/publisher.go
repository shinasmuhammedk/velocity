package userstream

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

func (p *Publisher) PublishOrderAccepted(
	userID int64,
	report ExecutionReport,
) {
	p.hub.Broadcast(
		userID,
		Message{
			Type: string(EventOrderAccepted),
			Data: report,
		},
	)
}

func (p *Publisher) PublishOrderRejected(
	userID int64,
	report ExecutionReport,
) {
	p.hub.Broadcast(
		userID,
		Message{
			Type: string(EventOrderRejected),
			Data: report,
		},
	)
}

func (p *Publisher) PublishOrderCancelled(
	userID int64,
	report ExecutionReport,
) {
	p.hub.Broadcast(
		userID,
		Message{
			Type: string(EventOrderCancelled),
			Data: report,
		},
	)
}

func (p *Publisher) PublishOrderFilled(
	userID int64,
	report ExecutionReport,
) {
	p.hub.Broadcast(
		userID,
		Message{
			Type: string(EventOrderFilled),
			Data: report,
		},
	)
}

func (p *Publisher) PublishBalanceUpdate(
	userID int64,
	update BalanceUpdate,
) {
	p.hub.Broadcast(
		userID,
		Message{
			Type: string(EventBalanceUpdated),
			Data: update,
		},
	)
}

func (p *Publisher) PublishPositionUpdate(
	userID int64,
	update PositionUpdate,
) {
	p.hub.Broadcast(
		userID,
		Message{
			Type: string(EventPositionUpdated),
			Data: update,
		},
	)
}

func (p *Publisher) PublishOrderModified(
	userID int64,
	report ExecutionReport,
) {
	p.hub.Broadcast(userID, Message{
		Type: string(EventOrderModified),
		Data: report,
	})
}

func (p *Publisher) PublishTradeExecuted(
	userID int64,
	trade TradeExecution,
) {
	p.hub.Broadcast(
		userID,
		Message{
			Type: string(EventTradeExecuted),
			Data: trade,
		},
	)
}

func (p *Publisher) PublishOrderPartiallyFilled(
	userID int64,
	report ExecutionReport,
) {
	p.hub.Broadcast(
		userID,
		Message{
			Type: string(EventOrderPartiallyFilled),
			Data: report,
		},
	)
}
