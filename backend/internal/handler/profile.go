package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/pulsechat/backend/internal/auth"
	"github.com/pulsechat/backend/internal/db"
)

// ProfileResponse is the full user profile including avatar.
type ProfileResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	CreatedAt string `json:"created_at"`
}

// UpdateProfileRequest payload for updating user details.
type UpdateProfileRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

// toProfileResponse converts a DB UserProfile to an API response.
func toProfileResponse(p *db.UserProfile) ProfileResponse {
	return ProfileResponse{
		ID:        p.ID,
		Username:  p.Username,
		Email:     p.Email,
		AvatarURL: p.AvatarURL,
		CreatedAt: p.CreatedAt.Format(time.RFC3339),
	}
}

// GetProfileHandler returns the authenticated user's full profile.
// GET /auth/profile
func GetProfileHandler(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, ok := auth.GetUserID(r.Context())
		if !ok {
			respondJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		profile, err := database.GetUserProfile(r.Context(), userID)
		if err != nil {
			slog.Error("failed to get user profile", "user_id", userID, "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Failed to load profile")
			return
		}

		_ = json.NewEncoder(w).Encode(toProfileResponse(profile))
	}
}

// UpdateProfileHandler updates the authenticated user's username and email.
// PUT /auth/profile
func UpdateProfileHandler(database *db.DB) http.HandlerFunc {
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

		profile, err := database.UpdateUserProfile(r.Context(), userID, req.Username, req.Email)
		if err != nil {
			slog.Error("failed to update user profile", "user_id", userID, "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Failed to update profile")
			return
		}

		// Also update localStorage user data via response
		_ = json.NewEncoder(w).Encode(toProfileResponse(profile))
	}
}

// UploadAvatarHandler uploads a new avatar image for the authenticated user.
// POST /auth/avatar — multipart/form-data with field "avatar"
func UploadAvatarHandler(database *db.DB, uploadDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, ok := auth.GetUserID(r.Context())
		if !ok {
			respondJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		// Limit to 2MB for avatars
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

		// Detect and validate content type
		buf := make([]byte, 512)
		n, _ := file.Read(buf)
		contentType := http.DetectContentType(buf[:n])

		if _, err := file.Seek(0, io.SeekStart); err != nil {
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		// Only image types allowed for avatars
		if !strings.HasPrefix(contentType, "image/") {
			respondJSONError(w, http.StatusBadRequest, "Only image files are allowed for avatars")
			return
		}

		// Remove old avatar file if it exists
		oldAvatarURL, _ := database.GetUserAvatarURL(r.Context(), userID)
		if oldAvatarURL != "" {
			oldPath := filepath.Join(uploadDir, filepath.Base(oldAvatarURL))
			_ = os.Remove(oldPath)
		}

		// Generate unique filename
		ext := filepath.Ext(header.Filename)
		if ext == "" {
			ext = ".png"
		}
		newFilename := "avatar_" + uuid.New().String() + strings.ToLower(ext)
		destPath := filepath.Join(uploadDir, newFilename)

		dst, err := os.Create(destPath)
		if err != nil {
			slog.Error("failed to create avatar file", "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Failed to save avatar")
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			slog.Error("failed to write avatar file", "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Failed to save avatar")
			return
		}

		avatarURL := "/uploads/" + newFilename

		// Update database
		profile, err := database.UpdateUserAvatar(r.Context(), userID, avatarURL)
		if err != nil {
			slog.Error("failed to update avatar in database", "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Failed to update avatar")
			return
		}

		slog.Info("avatar uploaded", "user_id", userID, "avatar_url", avatarURL)
		_ = json.NewEncoder(w).Encode(toProfileResponse(profile))
	}
}

// RemoveAvatarHandler clears the authenticated user's avatar.
// DELETE /auth/avatar
func RemoveAvatarHandler(database *db.DB, uploadDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, ok := auth.GetUserID(r.Context())
		if !ok {
			respondJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		// Remove old avatar file
		oldAvatarURL, _ := database.GetUserAvatarURL(r.Context(), userID)
		if oldAvatarURL != "" {
			oldPath := filepath.Join(uploadDir, filepath.Base(oldAvatarURL))
			_ = os.Remove(oldPath)
		}

		// Set avatar_url to empty
		profile, err := database.UpdateUserAvatar(r.Context(), userID, "")
		if err != nil {
			slog.Error("failed to remove avatar from database", "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Failed to remove avatar")
			return
		}

		slog.Info("avatar removed", "user_id", userID)
		_ = json.NewEncoder(w).Encode(toProfileResponse(profile))
	}
}
