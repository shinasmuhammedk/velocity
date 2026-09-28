package userstream

import (
	"sync"
	"velocity/internal/infrastructure/metrics"
)

// Subscriber is anything that can receive a broadcast message and be
// closed. *Client (a real websocket connection) satisfies this, and so
// can a lightweight in-memory test double, which is what lets
// integration tests assert on exactly what the settlement pipeline
// dispatches without standing up a websocket server.
type Subscriber interface {
	Send(message any) error
	Close() error
}

type Hub struct {
	clients map[int64]map[Subscriber]struct{}
	mu      sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[int64]map[Subscriber]struct{}),
	}
}

func (h *Hub) Subscribe(
	userID int64,
	client Subscriber,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.clients[userID]; !exists {
		h.clients[userID] = make(map[Subscriber]struct{})
	}

	h.clients[userID][client] = struct{}{}
}

func (h *Hub) Unsubscribe(
	userID int64,
	client Subscriber,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients, exists := h.clients[userID]
	if !exists {
		return
	}

	delete(clients, client)

	if len(clients) == 0 {
		delete(h.clients, userID)
	}
}

func (h *Hub) Broadcast(
	userID int64,
	message any,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients, exists := h.clients[userID]
	if !exists {
		return
	}

	for client := range clients {
		if err := client.Send(message); err != nil {
			eventType := "unknown"

			if msg, ok := message.(Message); ok {
				eventType = msg.Type
			}

			metrics.UserStreamDeliveryFailures.
				WithLabelValues(eventType).
				Inc()

			client.Close()
			delete(clients, client)
		}
	}

	if len(clients) == 0 {
		delete(h.clients, userID)
	}
}
