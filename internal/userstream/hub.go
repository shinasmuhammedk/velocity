package userstream

import "sync"

type Hub struct {
	clients map[int64]map[*Client]struct{}
	mu      sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[int64]map[*Client]struct{}),
	}
}

func (h *Hub) Subscribe(
	userID int64,
	client *Client,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.clients[userID]; !exists {
		h.clients[userID] = make(map[*Client]struct{})
	}

	h.clients[userID][client] = struct{}{}
}

func (h *Hub) Unsubscribe(
	userID int64,
	client *Client,
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
			client.Close()
			delete(clients, client)
		}
	}

	if len(clients) == 0 {
		delete(h.clients, userID)
	}
}
