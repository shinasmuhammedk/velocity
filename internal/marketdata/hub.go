package marketdata

import (
	"fmt"
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

	fmt.Println("SUBSCRIBED:", symbol)
	fmt.Println("CLIENT COUNT:", len(h.clients[symbol]))
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

	fmt.Println("BROADCAST:", symbol)
	fmt.Println("CLIENTS:", len(clients))

	for client := range clients {
		fmt.Println("Sending to client...")
		// client.Send(message)
		if err := client.Send(message); err != nil {
			delete(clients, client)
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
