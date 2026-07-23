package events

import "sync"

type Dispatcher struct {
	mu sync.RWMutex
    
    subscribers map[EventType][]Subscriber
}

func NewDispatcher () *Dispatcher{
    return &Dispatcher{
        subscribers: make(map[EventType][]Subscriber),
    }
}

func (d *Dispatcher) Subscribe(eventType EventType, sub Subscriber){
    d.mu.Lock()
    defer d.mu.Unlock()
    
    d.subscribers[eventType] = append(d.subscribers[eventType], sub)
}

func (d *Dispatcher) Publish(event Event){
    d.mu.RLock()
    
    subs := d.subscribers[event.Type()]
    
    d.mu.RUnlock()
    
    for _, sub := range subs {
		sub.Handle(event)
	}
}