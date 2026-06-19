package auth

import (
	"context"
	"testing"
	"time"
)

func TestGenerateAndValidateToken(t *testing.T) {
	secret := "test_secret_key"
	userID := "550e8400-e29b-41d4-a716-446655440000"
	username := "john_doe"
	email := "john@example.com"
	duration := 1 * time.Hour

	// 1. Generate Token
	token, err := GenerateToken(userID, username, email, secret, duration)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	if token == "" {
		t.Fatal("token should not be empty")
	}

	// 2. Validate Token
	claims, err := ValidateToken(token, secret)
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
	duration := -1 * time.Second // Expired immediately

	token, err := GenerateToken(userID, username, email, secret, duration)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = ValidateToken(token, secret)
	if err == nil {
		t.Fatal("expected error validating expired token, got nil")
	}

	if err != ErrExpiredToken {
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

	token, err := GenerateToken(userID, username, email, secret, duration)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = ValidateToken(token, wrongSecret)
	if err == nil {
		t.Fatal("expected error validating token with wrong secret, got nil")
	}

	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestContextAccessors(t *testing.T) {
	ctx := context.Background()

	// 1. Check unset context behavior
	_, exists := GetUserID(ctx)
	if exists {
		t.Error("expected GetUserID to return false for empty context")
	}

	// 2. Set values in context
	ctx = context.WithValue(ctx, userIDContextKey, "123")
	ctx = context.WithValue(ctx, usernameContextKey, "alice")
	ctx = context.WithValue(ctx, emailContextKey, "alice@example.com")

	id, exists := GetUserID(ctx)
	if !exists || id != "123" {
		t.Errorf("expected GetUserID to return '123', got '%s'", id)
	}

	user, exists := GetUsername(ctx)
	if !exists || user != "alice" {
		t.Errorf("expected GetUsername to return 'alice', got '%s'", user)
	}

	mail, exists := GetEmail(ctx)
	if !exists || mail != "alice@example.com" {
		t.Errorf("expected GetEmail to return 'alice@example.com', got '%s'", mail)
	}
}
