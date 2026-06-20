package service

import (
	"sync"

	"github.com/pulsechat/backend/internal/domain"
)

type PresenceService struct {
	mu    sync.RWMutex
	rooms map[string]map[string]domain.UserPresence
}

func NewPresenceService() *PresenceService {
	return &PresenceService{
		rooms: make(map[string]map[string]domain.UserPresence),
	}
}

func (s *PresenceService) Join(userID, roomID, username, avatarURL string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.rooms[roomID]; !exists {
		s.rooms[roomID] = make(map[string]domain.UserPresence)
	}

	s.rooms[roomID][userID] = domain.UserPresence{
		Username:  username,
		AvatarURL: avatarURL,
	}
}

func (s *PresenceService) Leave(userID, roomID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if userMap, exists := s.rooms[roomID]; exists {
		delete(userMap, userID)
		if len(userMap) == 0 {
			delete(s.rooms, roomID)
		}
	}
}

func (s *PresenceService) OnlineUsers(roomID string) []domain.OnlineUser {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userMap, exists := s.rooms[roomID]
	if !exists {
		return []domain.OnlineUser{}
	}

	users := make([]domain.OnlineUser, 0, len(userMap))
	for userID, presence := range userMap {
		users = append(users, domain.OnlineUser{
			UserID:    userID,
			Username:  presence.Username,
			AvatarURL: presence.AvatarURL,
		})
	}

	return users
}

func (s *PresenceService) IsOnline(userID, roomID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if userMap, exists := s.rooms[roomID]; exists {
		_, online := userMap[userID]
		return online
	}

	return false
}
