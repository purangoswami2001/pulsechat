package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type contextKey string

const (
	userContextKey     contextKey = "user"
	userIDContextKey   contextKey = "user_id"
	usernameContextKey contextKey = "username"
	emailContextKey    contextKey = "email"
)

// AuthMiddleware intercepts HTTP requests to check JWT tokens in Authorization headers.
func AuthMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondWithError(w, http.StatusUnauthorized, "Missing authorization header")
				return
			}

			// Expect Header format: "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				respondWithError(w, http.StatusUnauthorized, "Invalid authorization header format")
				return
			}

			tokenStr := parts[1]
			claims, err := ValidateToken(tokenStr, secret)
			if err != nil {
				respondWithError(w, http.StatusUnauthorized, err.Error())
				return
			}

			// Inject claims metadata into request context
			ctx := r.Context()
			ctx = context.WithValue(ctx, userIDContextKey, claims.UserID)
			ctx = context.WithValue(ctx, usernameContextKey, claims.Username)
			ctx = context.WithValue(ctx, emailContextKey, claims.Email)
			ctx = context.WithValue(ctx, userContextKey, claims)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID extracts the user ID string from context.
func GetUserID(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(userIDContextKey).(string)
	return val, ok
}

// GetUsername extracts the username string from context.
func GetUsername(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(usernameContextKey).(string)
	return val, ok
}

// GetEmail extracts the email address string from context.
func GetEmail(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(emailContextKey).(string)
	return val, ok
}

// Helper to write JSON error messages
func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
