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
	userID string,
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
	userID string,
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
	userID string,
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
	userID string,
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
	userID string,
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
	userID string,
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
