package pubsub

import (
	"context"
	"log/slog"
	"sync"
)

// MemoryPubSub implements the PubSub interface using in-process channels.
// Suitable for single-instance deployments where no external broker is needed.
type MemoryPubSub struct {
	mu          sync.RWMutex
	subscribers map[string][]chan []byte // topic -> list of subscriber channels
	closed      bool
}

// NewMemoryPubSub creates a new in-memory pub/sub broker.
func NewMemoryPubSub() *MemoryPubSub {
	slog.Info("initializing in-memory PubSub driver")
	return &MemoryPubSub{
		subscribers: make(map[string][]chan []byte),
	}
}

// Publish fans out the payload to all subscribers of the given topic.
func (m *MemoryPubSub) Publish(_ context.Context, topic string, payload []byte) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return ErrPubSubClosed
	}

	subs, ok := m.subscribers[topic]
	if !ok {
		return nil // no subscribers — silently drop
	}

	// Copy payload to avoid data races if caller reuses the slice
	msg := make([]byte, len(payload))
	copy(msg, payload)

	for _, ch := range subs {
		select {
		case ch <- msg:
		default:
			// Subscriber channel full — drop message to avoid blocking
			slog.Warn("memory pubsub: dropping message for slow subscriber", "topic", topic)
		}
	}

	return nil
}

// Subscribe creates a new subscriber channel for the given topic.
// The returned channel receives messages published to that topic.
// The channel is closed when the context is cancelled or Close() is called.
func (m *MemoryPubSub) Subscribe(ctx context.Context, topic string) (<-chan []byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil, ErrPubSubClosed
	}

	ch := make(chan []byte, 256)
	m.subscribers[topic] = append(m.subscribers[topic], ch)

	slog.Debug("memory pubsub: new subscription", "topic", topic, "total_subs", len(m.subscribers[topic]))

	if ctx != nil {
		go func() {
			<-ctx.Done()
			m.Unsubscribe(topic, ch)
		}()
	}

	return ch, nil
}

// Unsubscribe removes a specific subscriber channel from a topic.
func (m *MemoryPubSub) Unsubscribe(topic string, target <-chan []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	subs, ok := m.subscribers[topic]
	if !ok {
		return
	}

	for i, ch := range subs {
		if ch == target {
			// Remove from slice (swap with last element)
			subs[i] = subs[len(subs)-1]
			subs[len(subs)-1] = nil
			m.subscribers[topic] = subs[:len(subs)-1]
			close(ch)
			break
		}
	}

	// Clean up empty topic entry
	if len(m.subscribers[topic]) == 0 {
		delete(m.subscribers, topic)
	}
}

// Close shuts down the in-memory PubSub, closing all subscriber channels.
func (m *MemoryPubSub) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true

	for topic, subs := range m.subscribers {
		for _, ch := range subs {
			close(ch)
		}
		delete(m.subscribers, topic)
	}

	slog.Info("memory PubSub driver closed")
	return nil
}
