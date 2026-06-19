package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/pulsechat/backend/internal/auth"
	"github.com/pulsechat/backend/internal/config"
	"github.com/pulsechat/backend/internal/db"
)

// Request payloads
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Response payloads
type UserResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// RegisterHandler registers a new user, hashes their password, stores them, and returns a signed JWT token.
func RegisterHandler(database *db.DB, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSONError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		// Basic validation
		if req.Username == "" || req.Email == "" || req.Password == "" {
			respondJSONError(w, http.StatusBadRequest, "Username, email, and password are required")
			return
		}

		// Hash password using bcrypt
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			slog.Error("failed to hash password", "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		// Create user record parameters
		newUserID := uuid.New()
		userParams := db.CreateUserParams{
			ID:           newUserID,
			Username:     req.Username,
			Email:        req.Email,
			PasswordHash: string(hashedPassword),
		}

		// Save to database
		user, err := database.Queries.CreateUser(r.Context(), userParams)
		if err != nil {
			// Check for Postgres unique constraint violation
			// In pgx/v5, database errors usually wrap standard pgconn.PgError
			// (We log the error for diagnostics and return a user-friendly conflict code)
			slog.Warn("failed to create user in database", "error", err)
			respondJSONError(w, http.StatusConflict, "Username or email already exists")
			return
		}

		// Generate JWT Token
		token, err := auth.GenerateToken(
			user.ID.String(),
			user.Username,
			user.Email,
			cfg.JWTSecret,
			cfg.JWTExpirationHours,
		)
		if err != nil {
			slog.Error("failed to generate token", "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		// Respond with Token and User metadata
		resp := AuthResponse{
			Token: token,
			User: UserResponse{
				ID:        user.ID.String(),
				Username:  user.Username,
				Email:     user.Email,
				CreatedAt: user.CreatedAt.Time.Format(time.RFC3339),
			},
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// LoginHandler authenticates a user's password, and returns a signed JWT token if correct.
func LoginHandler(database *db.DB, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSONError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		if req.Email == "" || req.Password == "" {
			respondJSONError(w, http.StatusBadRequest, "Email and password are required")
			return
		}

		// Query user from database
		user, err := database.Queries.GetUserByEmail(r.Context(), req.Email)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				respondJSONError(w, http.StatusUnauthorized, "Invalid credentials")
				return
			}
			slog.Error("failed to query user email during login", "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		// Compare hashes
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
			respondJSONError(w, http.StatusUnauthorized, "Invalid credentials")
			return
		}

		// Generate JWT Token
		token, err := auth.GenerateToken(
			user.ID.String(),
			user.Username,
			user.Email,
			cfg.JWTSecret,
			cfg.JWTExpirationHours,
		)
		if err != nil {
			slog.Error("failed to generate login token", "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		// Respond with credentials
		resp := AuthResponse{
			Token: token,
			User: UserResponse{
				ID:        user.ID.String(),
				Username:  user.Username,
				Email:     user.Email,
				CreatedAt: user.CreatedAt.Time.Format(time.RFC3339),
			},
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// Helper to write JSON error messages
func respondJSONError(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}
