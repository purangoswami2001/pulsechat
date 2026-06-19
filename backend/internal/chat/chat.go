package chat

import (
	"context"
	"time"
)

// Message is our core chat message domain model.
type Message struct {
	ID             string    `json:"id"`
	RoomID         string    `json:"room_id"`
	SenderID       string    `json:"sender_id"`
	SenderName     string    `json:"sender_name"`     // Username resolved via DB joins
	SenderAvatarURL string   `json:"sender_avatar_url"` // Avatar URL resolved via DB joins
	Content        string    `json:"content"`
	AttachmentURL  string    `json:"attachment_url,omitempty"`
	AttachmentType string    `json:"attachment_type,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// MessageStore abstracts message persistence operations.
type MessageStore interface {
	Save(ctx context.Context, msg Message) error
	ListByRoom(ctx context.Context, roomID string, limit, offset int) ([]Message, error)
}

// ContactsStore abstracts fetching user contacts sharing rooms.
type ContactsStore interface {
	GetUserContacts(ctx context.Context, userID string) ([]string, error)
}

// OnlineUser holds connection info for a user currently online in a room.
type OnlineUser struct {
	UserID     string `json:"user_id"`
	Username   string `json:"username"`
	AvatarURL  string `json:"avatar_url"`
}

// PresenceManager tracks user connection states inside individual rooms.
type PresenceManager interface {
	Join(userID, roomID, username, avatarURL string)
	Leave(userID, roomID string)
	OnlineUsers(roomID string) []OnlineUser
	IsOnline(userID, roomID string) bool
}

// PubSub abstracts messaging broker operations for room synchronization.
type PubSub interface {
	Publish(ctx context.Context, topic string, payload []byte) error
	Subscribe(ctx context.Context, topic string) (<-chan []byte, error)
}
