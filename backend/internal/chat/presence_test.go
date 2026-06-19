package chat

import (
	"sync"
	"testing"
)

func TestPresenceManager_BasicFlow(t *testing.T) {
	presence := NewInMemoryPresence()
	roomID := "room-1"
	userA := "user-a"
	userB := "user-b"

	// 1. Join checks
	presence.Join(userA, roomID, "Alice")
	presence.Join(userB, roomID, "Bob")

	if !presence.IsOnline(userA, roomID) {
		t.Errorf("expected userA to be online")
	}
	if !presence.IsOnline(userB, roomID) {
		t.Errorf("expected userB to be online")
	}

	onlineUsers := presence.OnlineUsers(roomID)
	if len(onlineUsers) != 2 {
		t.Errorf("expected 2 online users, got %d", len(onlineUsers))
	}

	// Make sure both IDs exist
	hasUserA := false
	hasUserB := false
	for _, u := range onlineUsers {
		if u.UserID == userA && u.Username == "Alice" {
			hasUserA = true
		}
		if u.UserID == userB && u.Username == "Bob" {
			hasUserB = true
		}
	}

	if !hasUserA || !hasUserB {
		t.Errorf("online list did not contain all users: %v", onlineUsers)
	}

	// 2. Leave checks
	presence.Leave(userA, roomID)

	if presence.IsOnline(userA, roomID) {
		t.Errorf("expected userA to be offline")
	}

	onlineUsers = presence.OnlineUsers(roomID)
	if len(onlineUsers) != 1 || onlineUsers[0].UserID != userB || onlineUsers[0].Username != "Bob" {
		t.Errorf("expected userB to be the only online user, got %v", onlineUsers)
	}
}

func TestPresenceManager_ConcurrentAccess(t *testing.T) {
	presence := NewInMemoryPresence()
	roomID := "concurrency-room"
	workers := 100

	var wg sync.WaitGroup
	wg.Add(workers * 2)

	// Spin up concurrent joins
	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			userID := string(rune(id))
			presence.Join(userID, roomID, "User-"+userID)
		}(i)
	}

	// Spin up concurrent status checks
	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			userID := string(rune(id))
			_ = presence.IsOnline(userID, roomID)
			_ = presence.OnlineUsers(roomID)
		}(i)
	}

	wg.Wait()

	onlineCount := len(presence.OnlineUsers(roomID))
	if onlineCount != workers {
		t.Errorf("expected %d online users, got %d", workers, onlineCount)
	}
}
