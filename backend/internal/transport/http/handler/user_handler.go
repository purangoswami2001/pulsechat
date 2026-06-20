package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pulsechat/backend/internal/auth"
	"github.com/pulsechat/backend/internal/domain"
	"github.com/pulsechat/backend/internal/service"
)

type UserHandler struct {
	userService   *service.UserService
	uploadService *service.UploadService
}

func NewUserHandler(userService *service.UserService, uploadService *service.UploadService) *UserHandler {
	return &UserHandler{
		userService:   userService,
		uploadService: uploadService,
	}
}

type ProfileResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	CreatedAt string `json:"created_at"`
}

type UpdateProfileRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

func toProfileResponse(u *domain.User) ProfileResponse {
	return ProfileResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		AvatarURL: u.AvatarURL,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
	}
}

func (h *UserHandler) GetProfile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, ok := auth.GetUserID(r.Context())
		if !ok {
			respondJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		user, err := h.userService.GetProfile(r.Context(), userID)
		if err != nil {
			respondJSONError(w, http.StatusInternalServerError, "Failed to load profile")
			return
		}

		_ = json.NewEncoder(w).Encode(toProfileResponse(user))
	}
}

func (h *UserHandler) UpdateProfile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, ok := auth.GetUserID(r.Context())
		if !ok {
			respondJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		var req UpdateProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSONError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		if req.Username == "" || req.Email == "" {
			respondJSONError(w, http.StatusBadRequest, "Username and email are required")
			return
		}

		user, err := h.userService.UpdateProfile(r.Context(), userID, req.Username, req.Email)
		if err != nil {
			respondJSONError(w, http.StatusInternalServerError, "Failed to update profile")
			return
		}

		_ = json.NewEncoder(w).Encode(toProfileResponse(user))
	}
}

func (h *UserHandler) UploadAvatar() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, ok := auth.GetUserID(r.Context())
		if !ok {
			respondJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
		if err := r.ParseMultipartForm(2 << 20); err != nil {
			respondJSONError(w, http.StatusBadRequest, "Avatar too large. Maximum 2MB allowed.")
			return
		}

		file, header, err := r.FormFile("avatar")
		if err != nil {
			respondJSONError(w, http.StatusBadRequest, "File field 'avatar' is required")
			return
		}
		defer file.Close()

		buf := make([]byte, 512)
		n, _ := file.Read(buf)
		contentType := http.DetectContentType(buf[:n])

		if _, err := file.Seek(0, io.SeekStart); err != nil {
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		if !strings.HasPrefix(contentType, "image/") {
			respondJSONError(w, http.StatusBadRequest, "Only image files are allowed for avatars")
			return
		}

		// Check for old avatar to delete
		currentUser, err := h.userService.GetProfile(r.Context(), userID)
		if err == nil && currentUser.AvatarURL != "" {
			_ = h.uploadService.DeleteFile(currentUser.AvatarURL)
		}

		avatarURL, err := h.uploadService.SaveFile(header.Filename, contentType, file)
		if err != nil {
			respondJSONError(w, http.StatusInternalServerError, "Failed to save avatar")
			return
		}

		updatedUser, err := h.userService.UpdateAvatar(r.Context(), userID, avatarURL)
		if err != nil {
			respondJSONError(w, http.StatusInternalServerError, "Failed to update avatar")
			return
		}

		_ = json.NewEncoder(w).Encode(toProfileResponse(updatedUser))
	}
}

func (h *UserHandler) RemoveAvatar() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, ok := auth.GetUserID(r.Context())
		if !ok {
			respondJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		currentUser, err := h.userService.GetProfile(r.Context(), userID)
		if err == nil && currentUser.AvatarURL != "" {
			_ = h.uploadService.DeleteFile(currentUser.AvatarURL)
		}

		updatedUser, err := h.userService.UpdateAvatar(r.Context(), userID, "")
		if err != nil {
			respondJSONError(w, http.StatusInternalServerError, "Failed to remove avatar")
			return
		}

		_ = json.NewEncoder(w).Encode(toProfileResponse(updatedUser))
	}
}

func (h *UserHandler) SearchUsers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		callerID, ok := auth.GetUserID(r.Context())
		if !ok {
			respondJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		query := r.URL.Query().Get("q")
		limit := 10
		if lStr := r.URL.Query().Get("limit"); lStr != "" {
			if parsedL, err := strconv.Atoi(lStr); err == nil && parsedL > 0 {
				limit = parsedL
			}
		}

		users, err := h.userService.SearchUsers(r.Context(), query, callerID, limit)
		if err != nil {
			respondJSONError(w, http.StatusInternalServerError, "Failed to search users")
			return
		}

		_ = json.NewEncoder(w).Encode(users)
	}
}
