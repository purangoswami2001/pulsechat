package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/pulsechat/backend/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

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

func (h *AuthHandler) Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSONError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		if req.Username == "" || req.Email == "" || req.Password == "" {
			respondJSONError(w, http.StatusBadRequest, "Username, email, and password are required")
			return
		}

		res, err := h.authService.Register(r.Context(), req.Username, req.Email, req.Password)
		if err != nil {
			if errors.Is(err, service.ErrUserExists) {
				respondJSONError(w, http.StatusConflict, "Username or email already exists")
				return
			}
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		resp := AuthResponse{
			Token: res.Token,
			User: UserResponse{
				ID:        res.User.ID,
				Username:  res.User.Username,
				Email:     res.User.Email,
				CreatedAt: res.User.CreatedAt.Format(time.RFC3339),
			},
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func (h *AuthHandler) Login() http.HandlerFunc {
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

		res, err := h.authService.Login(r.Context(), req.Email, req.Password)
		if err != nil {
			if errors.Is(err, service.ErrInvalidCreds) {
				respondJSONError(w, http.StatusUnauthorized, "Invalid credentials")
				return
			}
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		resp := AuthResponse{
			Token: res.Token,
			User: UserResponse{
				ID:        res.User.ID,
				Username:  res.User.Username,
				Email:     res.User.Email,
				CreatedAt: res.User.CreatedAt.Format(time.RFC3339),
			},
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func respondJSONError(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
