package domain

import "time"

type Message struct {
	ID              string    `json:"id"`
	RoomID          string    `json:"room_id"`
	SenderID        string    `json:"sender_id"`
	SenderName      string    `json:"sender_name"`
	SenderAvatarURL string    `json:"sender_avatar_url"`
	Content         string    `json:"content"`
	AttachmentURL   string    `json:"attachment_url,omitempty"`
	AttachmentType  string    `json:"attachment_type,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}
