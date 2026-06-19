package pubsub

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// ErrPubSubClosed is returned when operations are attempted on a closed PubSub instance.
var ErrPubSubClosed = errors.New("pubsub: instance is closed")

// PubSub abstracts messaging broker operations for room synchronization.
// This mirrors the interface defined in the chat package for loose coupling.
type PubSub interface {
	Publish(ctx context.Context, topic string, payload []byte) error
	Subscribe(ctx context.Context, topic string) (<-chan []byte, error)
	io.Closer
}

// New creates a PubSub driver based on the given driver name.
// Supported drivers: "memory", "redis".
func New(driver string, redisURL string) (PubSub, error) {
	switch driver {
	case "memory":
		return NewMemoryPubSub(), nil
	case "redis":
		if redisURL == "" {
			return nil, fmt.Errorf("pubsub: REDIS_URL is required when using the redis driver")
		}
		return NewRedisPubSub(redisURL)
	default:
		return nil, fmt.Errorf("pubsub: unsupported driver %q (supported: memory, redis)", driver)
	}
}
