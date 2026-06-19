package chat

import (
	"testing"
	"time"
)

func TestTokenBucketRateLimiter_BurstAndThrottle(t *testing.T) {
	// Initialize Client structure
	client := &Client{
		tokens:     10.0,
		rate:       1.0,  // 1 token per second
		capacity:   10.0, // Max 10 tokens
		lastRefill: time.Now(),
	}

	// 1. First 10 requests should succeed immediately (Burst capacity)
	for i := 0; i < 10; i++ {
		if !client.Allow() {
			t.Errorf("request %d should have been allowed", i+1)
		}
	}

	// 2. The 11th request in quick succession should be blocked
	if client.Allow() {
		t.Error("11th request should have been blocked")
	}

	// 3. Wait 1.1 seconds -> should refill at least 1 token
	time.Sleep(1100 * time.Millisecond)

	if !client.Allow() {
		t.Error("request after waiting for refill should have been allowed")
	}

	// 4. Immediately following request should be blocked again
	if client.Allow() {
		t.Error("subsequent request without waiting should have been blocked")
	}
}
