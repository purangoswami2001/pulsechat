package pubsub_test

import (
	"context"
	"testing"
	"time"

	"github.com/pulsechat/backend/internal/pubsub"
)

func TestMemoryPubSub_ContextCancelUnsubscribes(t *testing.T) {
	ps := pubsub.NewMemoryPubSub()
	defer ps.Close()

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := ps.Subscribe(ctx, "user:abc")
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	ch2, err := ps.Subscribe(context.Background(), "user:abc")
	if err != nil {
		t.Fatalf("Subscribe ch2 failed: %v", err)
	}

	cancel()

	select {
	case _, open := <-ch:
		if open {
			t.Fatal("expected channel to close after context cancel")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for channel close after context cancel")
	}

	msg := []byte("hello")
	if err := ps.Publish(context.Background(), "user:abc", msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	select {
	case received := <-ch2:
		if string(received) != string(msg) {
			t.Errorf("got %q, want %q", received, msg)
		}
	case <-time.After(time.Second):
		t.Fatal("active subscriber did not receive message")
	}
}

func TestMemoryPubSub_PublishSubscribe(t *testing.T) {
	ps := pubsub.NewMemoryPubSub()
	defer ps.Close()

	ctx := context.Background()

	ch, err := ps.Subscribe(ctx, "room:abc")
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	msg := []byte(`{"type":"message.new","room_id":"abc","payload":{"content":"hello"}}`)
	if err := ps.Publish(ctx, "room:abc", msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	select {
	case received := <-ch:
		if string(received) != string(msg) {
			t.Errorf("got %q, want %q", received, msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestMemoryPubSub_MultipleSubs(t *testing.T) {
	ps := pubsub.NewMemoryPubSub()
	defer ps.Close()

	ctx := context.Background()

	ch1, _ := ps.Subscribe(ctx, "room:xyz")
	ch2, _ := ps.Subscribe(ctx, "room:xyz")

	msg := []byte("broadcast to all")
	if err := ps.Publish(ctx, "room:xyz", msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	for i, ch := range []<-chan []byte{ch1, ch2} {
		select {
		case received := <-ch:
			if string(received) != string(msg) {
				t.Errorf("sub %d: got %q, want %q", i, received, msg)
			}
		case <-time.After(time.Second):
			t.Fatalf("sub %d: timed out waiting for message", i)
		}
	}
}

func TestMemoryPubSub_NoSubsNoError(t *testing.T) {
	ps := pubsub.NewMemoryPubSub()
	defer ps.Close()

	ctx := context.Background()

	err := ps.Publish(ctx, "room:ghost", []byte("nobody home"))
	if err != nil {
		t.Fatalf("Publish to empty topic returned error: %v", err)
	}
}

func TestMemoryPubSub_Unsubscribe(t *testing.T) {
	ps := pubsub.NewMemoryPubSub()
	defer ps.Close()

	ctx := context.Background()

	ch, _ := ps.Subscribe(ctx, "room:leave")

	ps.Unsubscribe("room:leave", ch)

	_, open := <-ch
	if open {
		t.Fatal("expected channel to be closed after Unsubscribe")
	}
}

func TestMemoryPubSub_CloseClosesAllChannels(t *testing.T) {
	ps := pubsub.NewMemoryPubSub()

	ctx := context.Background()

	ch1, _ := ps.Subscribe(ctx, "room:a")
	ch2, _ := ps.Subscribe(ctx, "room:b")

	ps.Close()

	if _, open := <-ch1; open {
		t.Fatal("expected ch1 to be closed after Close()")
	}
	if _, open := <-ch2; open {
		t.Fatal("expected ch2 to be closed after Close()")
	}

	err := ps.Publish(ctx, "room:a", []byte("dead"))
	if err != pubsub.ErrPubSubClosed {
		t.Fatalf("expected ErrPubSubClosed, got %v", err)
	}

	_, err = ps.Subscribe(ctx, "room:a")
	if err != pubsub.ErrPubSubClosed {
		t.Fatalf("expected ErrPubSubClosed, got %v", err)
	}
}

func TestMemoryPubSub_IsolatedTopics(t *testing.T) {
	ps := pubsub.NewMemoryPubSub()
	defer ps.Close()

	ctx := context.Background()

	chA, _ := ps.Subscribe(ctx, "room:a")
	chB, _ := ps.Subscribe(ctx, "room:b")

	ps.Publish(ctx, "room:a", []byte("only for A"))

	select {
	case received := <-chA:
		if string(received) != "only for A" {
			t.Errorf("chA got unexpected: %q", received)
		}
	case <-time.After(time.Second):
		t.Fatal("chA: timed out")
	}

	select {
	case msg := <-chB:
		t.Fatalf("chB unexpectedly received: %q", msg)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestMemoryPubSub_PayloadIsolation(t *testing.T) {
	ps := pubsub.NewMemoryPubSub()
	defer ps.Close()

	ctx := context.Background()

	ch, _ := ps.Subscribe(ctx, "room:copy")

	original := []byte("original data")
	ps.Publish(ctx, "room:copy", original)

	original[0] = 'X'

	select {
	case received := <-ch:
		if received[0] == 'X' {
			t.Fatal("subscriber received mutated data — payload copy is broken")
		}
		if string(received) != "original data" {
			t.Errorf("got %q, want %q", received, "original data")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}
