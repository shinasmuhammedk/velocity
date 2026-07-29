package events

import (
	"fmt"
	"sync"
)

type Dispatcher struct {
	mu sync.RWMutex

	subscribers map[EventType][]Subscriber
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		subscribers: make(map[EventType][]Subscriber),
	}
}

func (d *Dispatcher) Subscribe(eventType EventType, sub Subscriber) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.subscribers[eventType] = append(d.subscribers[eventType], sub)

	fmt.Printf(
		"REGISTERED: %T for %s (total=%d)\n",
		sub,
		eventType,
		len(d.subscribers[eventType]),
	)
}

func (d *Dispatcher) Publish(event Event) {
	d.mu.RLock()
	subs := d.subscribers[event.Type()]
	d.mu.RUnlock()

	fmt.Println("PUBLISH:", event.Type())
	fmt.Println("Subscribers:", len(subs))

	for i, sub := range subs {
		fmt.Printf("Dispatching #%d -> %T\n", i+1, sub)
		sub.Handle(event)
	}
}
