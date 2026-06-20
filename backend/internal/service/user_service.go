package service

import (
	"context"

	"github.com/pulsechat/backend/internal/domain"
	"github.com/pulsechat/backend/internal/repository"
)

type UserService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) GetProfile(ctx context.Context, userID string) (*domain.User, error) {
	return s.userRepo.GetProfile(ctx, userID)
}

func (s *UserService) UpdateProfile(ctx context.Context, userID, username, email string) (*domain.User, error) {
	return s.userRepo.UpdateProfile(ctx, userID, username, email)
}

func (s *UserService) UpdateAvatar(ctx context.Context, userID, avatarURL string) (*domain.User, error) {
	return s.userRepo.UpdateAvatar(ctx, userID, avatarURL)
}

func (s *UserService) SearchUsers(ctx context.Context, query, excludeUserID string, limit int) ([]domain.User, error) {
	return s.userRepo.Search(ctx, query, excludeUserID, limit)
}

func (s *UserService) GetContacts(ctx context.Context, userID string) ([]string, error) {
	return s.userRepo.GetContacts(ctx, userID)
}
