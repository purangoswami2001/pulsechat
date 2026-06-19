package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/pulsechat/backend/internal/auth"
	"github.com/pulsechat/backend/internal/db"
)

// SearchUsersHandler finds users by username or email for invites and DMs.
func SearchUsersHandler(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, ok := auth.GetUserID(r.Context())
		if !ok {
			respondJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		query := r.URL.Query().Get("q")
		if len(query) < 1 {
			_ = json.NewEncoder(w).Encode([]db.UserSearchResult{})
			return
		}

		users, err := database.SearchUsers(r.Context(), query, userID, 10)
		if err != nil {
			slog.Error("failed to search users", "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		_ = json.NewEncoder(w).Encode(users)
	}
}
