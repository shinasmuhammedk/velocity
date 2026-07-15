package marketdata

import "sync"

type Hub struct {
	clients map[string]map[*Client]bool
	mu      sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]map[*Client]bool),
	}
}


func (h *Hub) Subscribe(
    symbol string,
    client *Client,
) {

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

    delete(h.clients[symbol], client)
}


func (h *Hub) Broadcast(
    symbol string,
    message any,
) {

    h.mu.RLock()
    defer h.mu.RUnlock()

    clients := h.clients[symbol]

    for client := range clients {
        client.Send(message)
    }
}