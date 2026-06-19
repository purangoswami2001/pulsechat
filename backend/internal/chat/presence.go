package chat

import (
	"sync"
)

// InMemoryPresence tracks room connection states in-memory.
type InMemoryPresence struct {
	mu    sync.RWMutex
	rooms map[string]map[string]UserPresence // map[roomID]map[userID]UserPresence
}

// UserPresence stores user info including avatar for presence tracking.
type UserPresence struct {
	Username  string
	AvatarURL string
}

// NewInMemoryPresence creates a thread-safe presence manager instance.
func NewInMemoryPresence() *InMemoryPresence {
	return &InMemoryPresence{
		rooms: make(map[string]map[string]UserPresence),
	}
}

// Join registers a user as active inside a channel.
func (p *InMemoryPresence) Join(userID, roomID, username, avatarURL string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.rooms[roomID]; !exists {
		p.rooms[roomID] = make(map[string]UserPresence)
	}

	p.rooms[roomID][userID] = UserPresence{
		Username:  username,
		AvatarURL: avatarURL,
	}
}

// Leave marks a user as inactive inside a channel.
func (p *InMemoryPresence) Leave(userID, roomID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if userMap, exists := p.rooms[roomID]; exists {
		delete(userMap, userID)
		// Clean up room map if empty
		if len(userMap) == 0 {
			delete(p.rooms, roomID)
		}
	}
}

// OnlineUsers lists all users currently online in a channel.
func (p *InMemoryPresence) OnlineUsers(roomID string) []OnlineUser {
	p.mu.RLock()
	defer p.mu.RUnlock()

	userMap, exists := p.rooms[roomID]
	if !exists {
		return []OnlineUser{}
	}

	users := make([]OnlineUser, 0, len(userMap))
	for userID, presence := range userMap {
		users = append(users, OnlineUser{
			UserID:    userID,
			Username:  presence.Username,
			AvatarURL: presence.AvatarURL,
		})
	}

	return users
}

// IsOnline checks if a specific user is active in a channel.
func (p *InMemoryPresence) IsOnline(userID, roomID string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if userMap, exists := p.rooms[roomID]; exists {
		_, online := userMap[userID]
		return online
	}

	return false
}
