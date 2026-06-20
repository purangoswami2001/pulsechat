package domain

type OnlineUser struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
}

type UserPresence struct {
	Username  string
	AvatarURL string
}
