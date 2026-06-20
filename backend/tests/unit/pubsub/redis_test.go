package pubsub_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/pulsechat/backend/internal/pubsub"
)

func TestRedisPubSub_PublishSubscribe(t *testing.T) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		t.Skip("REDIS_URL not set — skipping Redis integration test")
	}

	ps, err := pubsub.NewRedisPubSub(redisURL)
	if err != nil {
		t.Skipf("Redis not reachable, skipping integration test: %v", err)
	}
	defer ps.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch, err := ps.Subscribe(ctx, "test:room:redis")
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	msg := []byte(`{"type":"message.new","content":"hello from redis"}`)
	if err := ps.Publish(ctx, "test:room:redis", msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	select {
	case received := <-ch:
		if string(received) != string(msg) {
			t.Errorf("got %q, want %q", received, msg)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for Redis message")
	}
}

func TestRedisPubSub_ContextCancel(t *testing.T) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		t.Skip("REDIS_URL not set — skipping Redis integration test")
	}

	ps, err := pubsub.NewRedisPubSub(redisURL)
	if err != nil {
		t.Skipf("Redis not reachable, skipping integration test: %v", err)
	}
	defer ps.Close()

	ctx, cancel := context.WithCancel(context.Background())

	ch, err := ps.Subscribe(ctx, "test:room:cancel")
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	cancel()

	select {
	case _, open := <-ch:
		if open {
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for channel to close after context cancel")
	}
}

func TestNewRedisPubSub_InvalidURL(t *testing.T) {
	_, err := pubsub.NewRedisPubSub("not-a-valid-url")
	if err == nil {
		t.Fatal("expected error for invalid Redis URL")
	}
}
