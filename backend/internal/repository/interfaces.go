package repository

import (
	"context"

	"github.com/pulsechat/backend/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, id, username, email, passwordHash string) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	GetProfile(ctx context.Context, id string) (*domain.User, error)
	UpdateProfile(ctx context.Context, id, username, email string) (*domain.User, error)
	UpdateAvatar(ctx context.Context, id, avatarURL string) (*domain.User, error)
	Search(ctx context.Context, query, excludeUserID string, limit int) ([]domain.User, error)
	GetContacts(ctx context.Context, userID string) ([]string, error)
}

type RoomRepository interface {
	Create(ctx context.Context, id, name, roomType string) (*domain.Room, error)
	GetByID(ctx context.Context, id string) (*domain.Room, error)
	ListForUser(ctx context.Context, userID string) ([]domain.RoomWithMeta, error)
	IsMember(ctx context.Context, roomID, userID string) (bool, error)
	GetMembers(ctx context.Context, roomID string) ([]domain.RoomMemberProfile, error)
	AddMember(ctx context.Context, roomID, userID string, isAdmin bool) error
	RemoveMember(ctx context.Context, roomID, userID string) error
	IsAdmin(ctx context.Context, roomID, userID string) (bool, error)
	CountAdmins(ctx context.Context, roomID string) (int, error)
	Delete(ctx context.Context, roomID string) error
	FindDirectRoom(ctx context.Context, user1ID, user2ID string) (*domain.RoomWithMeta, error)
	CreateDirectRoom(ctx context.Context, user1ID, user2ID, otherUsername, otherAvatarURL string) (*domain.RoomWithMeta, error)
	GetMembersForMentions(ctx context.Context, roomID string) (string, string, []domain.MemberBrief, error)
}

type MessageRepository interface {
	Create(ctx context.Context, id, roomID, senderID, content string) (*domain.Message, error)
	CreateWithAttachment(ctx context.Context, id, roomID, senderID, content, attachmentURL, attachmentType string) error
	ListWithAttachments(ctx context.Context, roomID string, limit, offset int) ([]domain.Message, error)
}
