package domain

import "time"

type Room struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"` // "group", "direct"
	CreatedAt time.Time `json:"created_at"`
}

type RoomMember struct {
	RoomID   string    `json:"room_id"`
	UserID   string    `json:"user_id"`
	JoinedAt time.Time `json:"joined_at"`
	IsAdmin  bool      `json:"is_admin"`
}

type RoomWithMeta struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Type               string    `json:"type"`
	CreatedAt          time.Time `json:"created_at"`
	DisplayName        string    `json:"display_name,omitempty"`
	OtherUserID        string    `json:"other_user_id,omitempty"`
	OtherUserAvatarURL string    `json:"other_user_avatar_url,omitempty"`
	MemberCount        int       `json:"member_count,omitempty"`
}

type RoomMemberProfile struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	AvatarURL string    `json:"avatar_url"`
	JoinedAt  time.Time `json:"joined_at"`
	IsAdmin   bool      `json:"is_admin"`
}

type MemberBrief struct {
	UserID   string
	Username string
}
