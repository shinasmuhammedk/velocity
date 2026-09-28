package userstream

import (
	"errors"
	"testing"

	"velocity/internal/infrastructure/metrics"
)

type failingSubscriber struct {
	closed bool
}

func (f *failingSubscriber) Send(message any) error {
	return errors.New("simulated delivery failure")
}

func (f *failingSubscriber) Close() error {
	f.closed = true
	return nil
}

func TestHubBroadcast_DeliveryFailure(t *testing.T) {
	metrics.Register()

	hub := NewHub()

	client := &failingSubscriber{}
	hub.Subscribe(123, client)

	hub.Broadcast(123, Message{
		Type: "order_update",
		Data: map[string]any{
			"order_id": int64(1),
		},
	})

	if !client.closed {
		t.Fatal("expected failed subscriber to be closed")
	}

	clients := hub.clients[123]
	if len(clients) != 0 {
		t.Fatalf("expected failed subscriber to be removed, got %d clients", len(clients))
	}
}