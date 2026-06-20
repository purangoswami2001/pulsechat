package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/pulsechat/backend/internal/auth"
)

func TestGenerateAndValidateToken(t *testing.T) {
	secret := "test_secret_key"
	userID := "550e8400-e29b-41d4-a716-446655440000"
	username := "john_doe"
	email := "john@example.com"
	duration := 1 * time.Hour

	token, err := auth.GenerateToken(userID, username, email, secret, duration)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	if token == "" {
		t.Fatal("token should not be empty")
	}

	claims, err := auth.ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected userID %s, got %s", userID, claims.UserID)
	}
	if claims.Username != username {
		t.Errorf("expected username %s, got %s", username, claims.Username)
	}
	if claims.Email != email {
		t.Errorf("expected email %s, got %s", email, claims.Email)
	}
}

func TestValidateToken_Expired(t *testing.T) {
	secret := "test_secret_key"
	userID := "550e8400-e29b-41d4-a716-446655440000"
	username := "john_doe"
	email := "john@example.com"
	duration := -1 * time.Second

	token, err := auth.GenerateToken(userID, username, email, secret, duration)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = auth.ValidateToken(token, secret)
	if err == nil {
		t.Fatal("expected error validating expired token, got nil")
	}

	if err != auth.ErrExpiredToken {
		t.Errorf("expected ErrExpiredToken, got %v", err)
	}
}

func TestValidateToken_InvalidSecret(t *testing.T) {
	secret := "correct_secret"
	wrongSecret := "wrong_secret"
	userID := "550e8400-e29b-41d4-a716-446655440000"
	username := "john_doe"
	email := "john@example.com"
	duration := 1 * time.Hour

	token, err := auth.GenerateToken(userID, username, email, secret, duration)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = auth.ValidateToken(token, wrongSecret)
	if err == nil {
		t.Fatal("expected error validating token with wrong secret, got nil")
	}

	if err != auth.ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestContextAccessors(t *testing.T) {
	ctx := context.Background()

	_, exists := auth.GetUserID(ctx)
	if exists {
		t.Error("expected GetUserID to return false for empty context")
	}

	ctx = context.WithValue(ctx, auth.UserIDContextKey, "123")
	ctx = context.WithValue(ctx, auth.UsernameContextKey, "alice")
	ctx = context.WithValue(ctx, auth.EmailContextKey, "alice@example.com")

	id, exists := auth.GetUserID(ctx)
	if !exists || id != "123" {
		t.Errorf("expected GetUserID to return '123', got '%s'", id)
	}

	user, exists := auth.GetUsername(ctx)
	if !exists || user != "alice" {
		t.Errorf("expected GetUsername to return 'alice', got '%s'", user)
	}

	mail, exists := auth.GetEmail(ctx)
	if !exists || mail != "alice@example.com" {
		t.Errorf("expected GetEmail to return 'alice@example.com', got '%s'", mail)
	}
}
