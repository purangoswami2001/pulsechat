package mysql

import (
	"context"
	"errors"

	"github.com/pulsechat/backend/internal/domain"
)

type RoomRepository struct{}

func NewRoomRepository() *RoomRepository {
	return &RoomRepository{}
}

func (r *RoomRepository) Create(ctx context.Context, id, name, roomType string) (*domain.Room, error) {
	return nil, errors.New("mysql repository is not implemented")
}

func (r *RoomRepository) GetByID(ctx context.Context, id string) (*domain.Room, error) {
	return nil, errors.New("mysql repository is not implemented")
}

func (r *RoomRepository) ListForUser(ctx context.Context, userID string) ([]domain.RoomWithMeta, error) {
	return nil, errors.New("mysql repository is not implemented")
}

func (r *RoomRepository) IsMember(ctx context.Context, roomID, userID string) (bool, error) {
	return false, errors.New("mysql repository is not implemented")
}

func (r *RoomRepository) GetMembers(ctx context.Context, roomID string) ([]domain.RoomMemberProfile, error) {
	return nil, errors.New("mysql repository is not implemented")
}

func (r *RoomRepository) AddMember(ctx context.Context, roomID, userID string, isAdmin bool) error {
	return errors.New("mysql repository is not implemented")
}

func (r *RoomRepository) RemoveMember(ctx context.Context, roomID, userID string) error {
	return errors.New("mysql repository is not implemented")
}

func (r *RoomRepository) IsAdmin(ctx context.Context, roomID, userID string) (bool, error) {
	return false, errors.New("mysql repository is not implemented")
}

func (r *RoomRepository) CountAdmins(ctx context.Context, roomID string) (int, error) {
	return 0, errors.New("mysql repository is not implemented")
}

func (r *RoomRepository) Delete(ctx context.Context, roomID string) error {
	return errors.New("mysql repository is not implemented")
}

func (r *RoomRepository) FindDirectRoom(ctx context.Context, user1ID, user2ID string) (*domain.RoomWithMeta, error) {
	return nil, errors.New("mysql repository is not implemented")
}

func (r *RoomRepository) CreateDirectRoom(ctx context.Context, user1ID, user2ID, otherUsername, otherAvatarURL string) (*domain.RoomWithMeta, error) {
	return nil, errors.New("mysql repository is not implemented")
}

func (r *RoomRepository) GetMembersForMentions(ctx context.Context, roomID string) (string, string, []domain.MemberBrief, error) {
	return "", "", nil, errors.New("mysql repository is not implemented")
}
