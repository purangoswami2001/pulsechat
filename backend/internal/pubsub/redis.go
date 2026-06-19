package pubsub

import (
	"context"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

// RedisPubSub implements the PubSub interface using Redis Pub/Sub channels.
// This enables broadcasting across multiple server instances for horizontal scaling.
type RedisPubSub struct {
	client *redis.Client
}

// NewRedisPubSub creates a Redis-backed PubSub driver.
// It parses the provided Redis URL and pings the server to validate the connection.
func NewRedisPubSub(redisURL string) (*RedisPubSub, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opts)

	// Verify connectivity
	if err := client.Ping(context.Background()).Err(); err != nil {
		client.Close()
		return nil, err
	}

	slog.Info("initializing Redis PubSub driver", "url", redisURL)
	return &RedisPubSub{client: client}, nil
}

// Publish sends a payload to the specified Redis channel topic.
func (r *RedisPubSub) Publish(ctx context.Context, topic string, payload []byte) error {
	return r.client.Publish(ctx, topic, payload).Err()
}

// Subscribe creates a Redis subscription for the given topic.
// It spawns a background goroutine that reads from the Redis subscription
// and forwards messages to the returned channel.
// The channel is closed when the context is cancelled.
func (r *RedisPubSub) Subscribe(ctx context.Context, topic string) (<-chan []byte, error) {
	sub := r.client.Subscribe(ctx, topic)

	// Verify the subscription is active
	if _, err := sub.Receive(ctx); err != nil {
		sub.Close()
		return nil, err
	}

	ch := make(chan []byte, 256)

	go func() {
		defer close(ch)
		defer sub.Close()

		msgCh := sub.Channel()
		for {
			select {
			case <-ctx.Done():
				slog.Debug("redis pubsub: subscription cancelled", "topic", topic)
				return
			case msg, ok := <-msgCh:
				if !ok {
					slog.Debug("redis pubsub: subscription channel closed", "topic", topic)
					return
				}
				select {
				case ch <- []byte(msg.Payload):
				default:
					slog.Warn("redis pubsub: dropping message for slow consumer", "topic", topic)
				}
			}
		}
	}()

	slog.Debug("redis pubsub: new subscription", "topic", topic)
	return ch, nil
}

// Close shuts down the Redis client connection.
func (r *RedisPubSub) Close() error {
	slog.Info("Redis PubSub driver closing")
	return r.client.Close()
}
