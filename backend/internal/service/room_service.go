package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/pulsechat/backend/internal/domain"
	"github.com/pulsechat/backend/internal/repository"
)

type RoomService struct {
	roomRepo repository.RoomRepository
	userRepo repository.UserRepository
}

func NewRoomService(roomRepo repository.RoomRepository, userRepo repository.UserRepository) *RoomService {
	return &RoomService{
		roomRepo: roomRepo,
		userRepo: userRepo,
	}
}

func (s *RoomService) CreateRoom(ctx context.Context, name, roomType string, creatorID string, memberIDs []string) (*domain.Room, error) {
	roomID := uuid.New().String()
	room, err := s.roomRepo.Create(ctx, roomID, name, roomType)
	if err != nil {
		return nil, err
	}

	// Add creator as admin
	if err := s.roomRepo.AddMember(ctx, roomID, creatorID, true); err != nil {
		return nil, err
	}

	// Add other members
	seen := map[string]bool{creatorID: true}
	for _, mid := range memberIDs {
		if mid == "" || seen[mid] {
			continue
		}
		seen[mid] = true
		_ = s.roomRepo.AddMember(ctx, roomID, mid, false)
	}

	return room, nil
}

func (s *RoomService) GetByID(ctx context.Context, id string) (*domain.Room, error) {
	return s.roomRepo.GetByID(ctx, id)
}

func (s *RoomService) ListRoomsForUser(ctx context.Context, userID string) ([]domain.RoomWithMeta, error) {
	return s.roomRepo.ListForUser(ctx, userID)
}

func (s *RoomService) IsRoomMember(ctx context.Context, roomID, userID string) (bool, error) {
	return s.roomRepo.IsMember(ctx, roomID, userID)
}

func (s *RoomService) GetRoomMembers(ctx context.Context, roomID string) ([]domain.RoomMemberProfile, error) {
	return s.roomRepo.GetMembers(ctx, roomID)
}

func (s *RoomService) AddRoomMember(ctx context.Context, roomID, userID string, isAdmin bool) error {
	return s.roomRepo.AddMember(ctx, roomID, userID, isAdmin)
}

func (s *RoomService) RemoveRoomMember(ctx context.Context, roomID, userID string) error {
	return s.roomRepo.RemoveMember(ctx, roomID, userID)
}

func (s *RoomService) IsGroupAdmin(ctx context.Context, roomID, userID string) (bool, error) {
	return s.roomRepo.IsAdmin(ctx, roomID, userID)
}

func (s *RoomService) CountGroupAdmins(ctx context.Context, roomID string) (int, error) {
	return s.roomRepo.CountAdmins(ctx, roomID)
}

func (s *RoomService) DeleteGroup(ctx context.Context, roomID string) error {
	return s.roomRepo.Delete(ctx, roomID)
}

func (s *RoomService) GetOrCreateDirectRoom(ctx context.Context, callerID, targetUserID string) (*domain.RoomWithMeta, error) {
	otherUser, err := s.userRepo.GetProfile(ctx, targetUserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	existing, err := s.roomRepo.FindDirectRoom(ctx, callerID, targetUserID)
	if err == nil && existing != nil {
		existing.DisplayName = otherUser.Username
		existing.OtherUserID = otherUser.ID
		existing.OtherUserAvatarURL = otherUser.AvatarURL
		return existing, nil
	}

	return s.roomRepo.CreateDirectRoom(ctx, callerID, targetUserID, otherUser.Username, otherUser.AvatarURL)
}

func (s *RoomService) GetRoomMembersForMentions(ctx context.Context, roomID string) (string, string, []domain.MemberBrief, error) {
	return s.roomRepo.GetMembersForMentions(ctx, roomID)
}
