package marketdata

import (
	"sync"
)

type Hub struct {
	clients map[string]map[*Client]bool
	mu      sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]map[*Client]bool),
	}
}

func (h *Hub) Subscribe(symbol string, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.clients[symbol]; !exists {
		h.clients[symbol] = make(map[*Client]bool)
	}

	h.clients[symbol][client] = true
}

func (h *Hub) Unsubscribe(
	symbol string,
	client *Client,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients, exists := h.clients[symbol]
	if !exists {
		return
	}

	delete(clients, client)

	if len(clients) == 0 {
		delete(h.clients, symbol)
	}
}

func (h *Hub) Broadcast(
	symbol string,
	message any,
) {
	// Take a snapshot of the subscribed clients.
	h.mu.RLock()

	clients := h.clients[symbol]
	snapshot := make([]*Client, 0, len(clients))

	for client := range clients {
		snapshot = append(snapshot, client)
	}

	h.mu.RUnlock()

	// Network I/O happens outside the Hub lock.
	for _, client := range snapshot {
		if err := client.Send(message); err != nil {
			h.Unsubscribe(symbol, client)
		}
	}
}

func (h *Hub) RemoveClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for symbol, clients := range h.clients {
		delete(clients, client)

		if len(clients) == 0 {
			delete(h.clients, symbol)
		}
	}
}
