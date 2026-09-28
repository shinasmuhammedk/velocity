package testhelpers

import (
	"sync"

	"velocity/internal/userstream"
)

// FakeSubscriber is a userstream.Subscriber test double. It records every
// message broadcast to it in memory instead of writing to a real
// websocket connection, which lets integration tests assert on exactly
// which events a code path dispatched, to whom, and with what payload -
// without standing up a websocket server.
//
// Settlement (and every other dispatcher) calls Hub.Broadcast
// synchronously on the calling goroutine, so messages recorded by a
// FakeSubscriber are visible to the test as soon as the call that
// triggered them (e.g. service.Settle) returns. The mutex below exists
// only to make it safe to also use a FakeSubscriber from concurrency
// tests that call into the dispatcher from multiple goroutines at once.
type FakeSubscriber struct {
	mu       sync.Mutex
	messages []userstream.Message
	closed   bool
}

// NewFakeSubscriber returns a ready-to-subscribe FakeSubscriber.
func NewFakeSubscriber() *FakeSubscriber {
	return &FakeSubscriber{}
}

// Send implements userstream.Subscriber.
func (f *FakeSubscriber) Send(message any) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	msg, ok := message.(userstream.Message)
	if !ok {
		// Every publisher method in this codebase broadcasts a
		// userstream.Message. If that ever changes, fail loudly in
		// tests rather than silently dropping the message.
		panic("testhelpers: FakeSubscriber received a non-userstream.Message value")
	}

	f.messages = append(f.messages, msg)

	return nil
}

// Close implements userstream.Subscriber.
func (f *FakeSubscriber) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.closed = true

	return nil
}

// Messages returns a snapshot of every message received so far, in the
// order they were dispatched.
func (f *FakeSubscriber) Messages() []userstream.Message {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]userstream.Message, len(f.messages))
	copy(out, f.messages)

	return out
}

// MessagesOfType returns, in dispatch order, only the messages whose
// Type matches the given userstream.EventType.
func (f *FakeSubscriber) MessagesOfType(eventType userstream.EventType) []userstream.Message {
	var out []userstream.Message

	for _, msg := range f.Messages() {
		if msg.Type == string(eventType) {
			out = append(out, msg)
		}
	}

	return out
}

// Count returns the number of messages received so far.
func (f *FakeSubscriber) Count() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.messages)
}

// Closed reports whether the hub called Close on this subscriber (e.g.
// after a failed Send).
func (f *FakeSubscriber) Closed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.closed
}