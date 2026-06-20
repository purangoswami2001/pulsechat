package mysql

import (
	"context"
	"errors"

	"github.com/pulsechat/backend/internal/domain"
)

type UserRepository struct{}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

func (r *UserRepository) Create(ctx context.Context, id, username, email, passwordHash string) (*domain.User, error) {
	return nil, errors.New("mysql repository is not implemented")
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return nil, errors.New("mysql repository is not implemented")
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return nil, errors.New("mysql repository is not implemented")
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	return nil, errors.New("mysql repository is not implemented")
}

func (r *UserRepository) GetProfile(ctx context.Context, id string) (*domain.User, error) {
	return nil, errors.New("mysql repository is not implemented")
}

func (r *UserRepository) UpdateProfile(ctx context.Context, id, username, email string) (*domain.User, error) {
	return nil, errors.New("mysql repository is not implemented")
}

func (r *UserRepository) UpdateAvatar(ctx context.Context, id, avatarURL string) (*domain.User, error) {
	return nil, errors.New("mysql repository is not implemented")
}

func (r *UserRepository) Search(ctx context.Context, query, excludeUserID string, limit int) ([]domain.User, error) {
	return nil, errors.New("mysql repository is not implemented")
}

func (r *UserRepository) GetContacts(ctx context.Context, userID string) ([]string, error) {
	return nil, errors.New("mysql repository is not implemented")
}
