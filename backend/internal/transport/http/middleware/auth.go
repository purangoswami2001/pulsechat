package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pulsechat/backend/internal/auth"
)

func AuthMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondWithError(w, http.StatusUnauthorized, "Missing authorization header")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				respondWithError(w, http.StatusUnauthorized, "Invalid authorization header format")
				return
			}

			tokenStr := parts[1]
			claims, err := auth.ValidateToken(tokenStr, secret)
			if err != nil {
				respondWithError(w, http.StatusUnauthorized, err.Error())
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, auth.UserIDContextKey, claims.UserID)
			ctx = context.WithValue(ctx, auth.UsernameContextKey, claims.Username)
			ctx = context.WithValue(ctx, auth.EmailContextKey, claims.Email)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
