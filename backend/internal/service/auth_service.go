package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/pulsechat/backend/internal/auth"
	"github.com/pulsechat/backend/internal/domain"
	"github.com/pulsechat/backend/internal/repository"
)

var (
	ErrUserExists       = errors.New("username or email already exists")
	ErrInvalidCreds     = errors.New("invalid credentials")
	ErrUserNotFound     = errors.New("user not found")
)

type AuthService struct {
	userRepo             repository.UserRepository
	jwtSecret            string
	jwtExpirationHours   time.Duration
}

func NewAuthService(userRepo repository.UserRepository, jwtSecret string, jwtExpirationHours time.Duration) *AuthService {
	return &AuthService{
		userRepo:           userRepo,
		jwtSecret:          jwtSecret,
		jwtExpirationHours: jwtExpirationHours,
	}
}

type AuthResult struct {
	Token string
	User  *domain.User
}

func (s *AuthService) Register(ctx context.Context, username, email, password string) (*AuthResult, error) {
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}

	newID := uuid.New().String()
	user, err := s.userRepo.Create(ctx, newID, username, email, hashedPassword)
	if err != nil {
		return nil, ErrUserExists
	}

	token, err := auth.GenerateToken(user.ID, user.Username, user.Email, s.jwtSecret, s.jwtExpirationHours)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		Token: token,
		User:  user,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCreds
	}

	if !auth.CheckPasswordHash(password, user.PasswordHash) {
		return nil, ErrInvalidCreds
	}

	token, err := auth.GenerateToken(user.ID, user.Username, user.Email, s.jwtSecret, s.jwtExpirationHours)
	if err != nil {
		return nil, err
	}

	// Remove password hash from domain model response
	user.PasswordHash = ""

	return &AuthResult{
		Token: token,
		User:  user,
	}, nil
}
