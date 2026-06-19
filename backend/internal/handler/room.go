package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/pulsechat/backend/internal/auth"
	"github.com/pulsechat/backend/internal/chat"
	"github.com/pulsechat/backend/internal/db"
)

type CreateRoomRequest struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"` // "group"
	MemberIDs []string `json:"member_ids"`
}

type AddMemberRequest struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

type CreateDirectRequest struct {
	UserID string `json:"user_id"`
}

// CreateRoomHandler handles group creation and records the creator as admin.
func CreateRoomHandler(database *db.DB, hub *chat.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		creatorIDStr, ok := auth.GetUserID(r.Context())
		if !ok {
			respondJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		creatorName, _ := auth.GetUsername(r.Context())

		creatorID, err := uuid.Parse(creatorIDStr)
		if err != nil {
			respondJSONError(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}

		var req CreateRoomRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSONError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		if req.Name == "" {
			respondJSONError(w, http.StatusBadRequest, "Group name is required")
			return
		}

		if req.Type == "" {
			req.Type = "group"
		}

		if req.Type != "group" && req.Type != "private" {
			respondJSONError(w, http.StatusBadRequest, "Only group chats are supported")
			return
		}

		// Store as group (private kept for backward compatibility during migration)
		if req.Type == "private" {
			req.Type = "group"
		}

		tx, err := database.Pool.Begin(r.Context())
		if err != nil {
			slog.Error("failed to start room creation transaction", "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		defer func() {
			_ = tx.Rollback(r.Context())
		}()

		qtx := database.Queries.WithTx(tx)

		newRoomID := uuid.New()
		roomParams := db.CreateRoomParams{
			ID:   newRoomID,
			Name: req.Name,
			Type: req.Type,
		}

		room, err := qtx.CreateRoom(r.Context(), roomParams)
		if err != nil {
			slog.Error("failed to create group", "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		_, err = tx.Exec(r.Context(),
			`INSERT INTO room_members (room_id, user_id, joined_at, is_admin) VALUES ($1, $2, CURRENT_TIMESTAMP, true)`,
			newRoomID, creatorID)
		if err != nil {
			slog.Error("failed to register creator as admin", "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		invitedIDs := make([]string, 0)
		seen := map[string]bool{creatorIDStr: true}
		for _, memberIDStr := range req.MemberIDs {
			if memberIDStr == "" || seen[memberIDStr] {
				continue
			}
			seen[memberIDStr] = true

			memberID, parseErr := uuid.Parse(memberIDStr)
			if parseErr != nil {
				respondJSONError(w, http.StatusBadRequest, "Invalid member user ID")
				return
			}

			_, err = tx.Exec(r.Context(),
				`INSERT INTO room_members (room_id, user_id, joined_at, is_admin) VALUES ($1, $2, CURRENT_TIMESTAMP, false)`,
				newRoomID, memberID)
			if err != nil {
				slog.Error("failed to register room membership", "error", err)
				respondJSONError(w, http.StatusInternalServerError, "Internal server error")
				return
			}
			invitedIDs = append(invitedIDs, memberIDStr)
		}

		if err := tx.Commit(r.Context()); err != nil {
			slog.Error("failed to commit transaction", "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		for _, memberID := range invitedIDs {
			hub.NotifyGroupInvite(memberID, chat.GroupInviteNotification{
				GroupID:     newRoomID.String(),
				GroupName:   req.Name,
				InviterID:   creatorIDStr,
				InviterName: creatorName,
			})
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(room)
	}
}

// ListRoomsHandler lists rooms visible to the authenticated user.
func ListRoomsHandler(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, ok := auth.GetUserID(r.Context())
		if !ok {
			respondJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		rooms, err := database.ListRoomsForUser(r.Context(), userID)
		if err != nil {
			slog.Error("failed to list rooms for user", "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		_ = json.NewEncoder(w).Encode(rooms)
	}
}

// CreateDirectRoomHandler finds or creates a 1-on-1 chat with another user.
func CreateDirectRoomHandler(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		callerID, ok := auth.GetUserID(r.Context())
		if !ok {
			respondJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		var req CreateDirectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSONError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		if req.UserID == "" {
			respondJSONError(w, http.StatusBadRequest, "user_id is required")
			return
		}

		if req.UserID == callerID {
			respondJSONError(w, http.StatusBadRequest, "Cannot start a direct chat with yourself")
			return
		}

		otherUser, err := database.GetUserProfile(r.Context(), req.UserID)
		if err != nil {
			respondJSONError(w, http.StatusNotFound, "User not found")
			return
		}

		existing, err := database.FindDirectRoom(r.Context(), callerID, req.UserID)
		if err == nil && existing != nil {
			existing.DisplayName = otherUser.Username
			existing.OtherUserID = otherUser.ID
			existing.OtherUserAvatarURL = otherUser.AvatarURL
			_ = json.NewEncoder(w).Encode(existing)
			return
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			slog.Error("failed to find direct room", "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		room, err := database.CreateDirectRoom(r.Context(), callerID, req.UserID, otherUser.Username, otherUser.AvatarURL)
		if err != nil {
			slog.Error("failed to create direct room", "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(room)
	}
}

// GetRoomMembersHandler returns persisted members of a room.
func GetRoomMembersHandler(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, ok := auth.GetUserID(r.Context())
		if !ok {
			respondJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		roomID := r.PathValue("roomID")
		if roomID == "" {
			respondJSONError(w, http.StatusBadRequest, "Room ID is required")
			return
		}

		canAccess, err := database.CanAccessRoom(r.Context(), roomID, userID)
		if err != nil || !canAccess {
			respondJSONError(w, http.StatusForbidden, "Access denied")
			return
		}

		members, err := database.GetRoomMembersWithAvatar(r.Context(), roomID)
		if err != nil {
			slog.Error("failed to get room members", "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		_ = json.NewEncoder(w).Encode(members)
	}
}

// AddRoomMemberHandler invites a user to a group (admin only).
func AddRoomMemberHandler(database *db.DB, hub *chat.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		callerID, ok := auth.GetUserID(r.Context())
		if !ok {
			respondJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		callerName, _ := auth.GetUsername(r.Context())

		roomID := r.PathValue("roomID")
		if roomID == "" {
			respondJSONError(w, http.StatusBadRequest, "Room ID is required")
			return
		}

		roomUUID, err := uuid.Parse(roomID)
		if err != nil {
			respondJSONError(w, http.StatusBadRequest, "Invalid room ID")
			return
		}

		room, err := database.Queries.GetRoomByID(r.Context(), roomUUID)
		if err != nil {
			respondJSONError(w, http.StatusNotFound, "Room not found")
			return
		}

		if room.Type != "group" && room.Type != "private" {
			respondJSONError(w, http.StatusBadRequest, "Members can only be added to group chats")
			return
		}

		isAdmin, err := database.IsGroupAdmin(r.Context(), roomID, callerID)
		if err != nil || !isAdmin {
			respondJSONError(w, http.StatusForbidden, "Only group admins can add members")
			return
		}

		var req AddMemberRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSONError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		targetUserID := req.UserID
		if targetUserID == "" && req.Username != "" {
			user, lookupErr := database.GetUserByUsername(r.Context(), req.Username)
			if lookupErr != nil {
				respondJSONError(w, http.StatusNotFound, "User not found")
				return
			}
			targetUserID = user.ID
		}

		if targetUserID == "" {
			respondJSONError(w, http.StatusBadRequest, "user_id or username is required")
			return
		}

		if err := database.AddRoomMember(r.Context(), roomID, targetUserID, false); err != nil {
			slog.Error("failed to add room member", "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		hub.NotifyGroupInvite(targetUserID, chat.GroupInviteNotification{
			GroupID:     roomID,
			GroupName:   room.Name,
			InviterID:   callerID,
			InviterName: callerName,
		})

		members, err := database.GetRoomMembersWithAvatar(r.Context(), roomID)
		if err != nil {
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(members)
	}
}

// DeleteGroupHandler deletes a group (admin only).
func DeleteGroupHandler(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		callerID, ok := auth.GetUserID(r.Context())
		if !ok {
			respondJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		roomID := r.PathValue("roomID")
		if roomID == "" {
			respondJSONError(w, http.StatusBadRequest, "Room ID is required")
			return
		}

		roomUUID, err := uuid.Parse(roomID)
		if err != nil {
			respondJSONError(w, http.StatusBadRequest, "Invalid room ID")
			return
		}

		room, err := database.Queries.GetRoomByID(r.Context(), roomUUID)
		if err != nil {
			respondJSONError(w, http.StatusNotFound, "Group not found")
			return
		}

		if room.Type != "group" && room.Type != "private" && room.Type != "direct" {
			respondJSONError(w, http.StatusBadRequest, "Only groups and direct chats can be deleted")
			return
		}

		if room.Type == "group" || room.Type == "private" {
			isAdmin, err := database.IsGroupAdmin(r.Context(), roomID, callerID)
			if err != nil || !isAdmin {
				respondJSONError(w, http.StatusForbidden, "Only group admins can delete the group")
				return
			}
		} else if room.Type == "direct" {
			canAccess, err := database.CanAccessRoom(r.Context(), roomID, callerID)
			if err != nil || !canAccess {
				respondJSONError(w, http.StatusForbidden, "Only participants can delete this direct chat")
				return
			}
		}

		if err := database.DeleteGroup(r.Context(), roomID); err != nil {
			slog.Error("failed to delete group", "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// RemoveRoomMemberHandler removes a user from a group (admin only).
func RemoveRoomMemberHandler(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		callerID, ok := auth.GetUserID(r.Context())
		if !ok {
			respondJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		roomID := r.PathValue("roomID")
		targetUserID := r.PathValue("userID")
		if roomID == "" || targetUserID == "" {
			respondJSONError(w, http.StatusBadRequest, "Room ID and user ID are required")
			return
		}

		roomUUID, err := uuid.Parse(roomID)
		if err != nil {
			respondJSONError(w, http.StatusBadRequest, "Invalid room ID")
			return
		}

		room, err := database.Queries.GetRoomByID(r.Context(), roomUUID)
		if err != nil {
			respondJSONError(w, http.StatusNotFound, "Group not found")
			return
		}

		if room.Type != "group" && room.Type != "private" {
			respondJSONError(w, http.StatusBadRequest, "Members can only be removed from group chats")
			return
		}

		isAdmin, err := database.IsGroupAdmin(r.Context(), roomID, callerID)
		if err != nil || !isAdmin {
			respondJSONError(w, http.StatusForbidden, "Only group admins can remove members")
			return
		}

		if targetUserID == callerID {
			respondJSONError(w, http.StatusBadRequest, "Admins cannot remove themselves")
			return
		}

		targetIsAdmin, err := database.IsGroupAdmin(r.Context(), roomID, targetUserID)
		if err == nil && targetIsAdmin {
			adminCount, countErr := database.CountGroupAdmins(r.Context(), roomID)
			if countErr == nil && adminCount <= 1 {
				respondJSONError(w, http.StatusBadRequest, "Cannot remove the only group admin")
				return
			}
		}

		if err := database.RemoveRoomMember(r.Context(), roomID, targetUserID); err != nil {
			slog.Error("failed to remove room member", "error", err)
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		members, err := database.GetRoomMembersWithAvatar(r.Context(), roomID)
		if err != nil {
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		_ = json.NewEncoder(w).Encode(members)
	}
}

// GetRoomMessagesHandler retrieves paginated message history including attachments.
func GetRoomMessagesHandler(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, ok := auth.GetUserID(r.Context())
		if !ok {
			respondJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		roomID := r.PathValue("roomID")
		if roomID == "" {
			respondJSONError(w, http.StatusBadRequest, "Room ID is required")
			return
		}

		canAccess, err := database.CanAccessRoom(r.Context(), roomID, userID)
		if err != nil || !canAccess {
			respondJSONError(w, http.StatusForbidden, "Access denied")
			return
		}

		limit := 50
		offset := 0

		if lStr := r.URL.Query().Get("limit"); lStr != "" {
			if parsedL, parseErr := strconv.Atoi(lStr); parseErr == nil && parsedL > 0 {
				limit = parsedL
			}
		}

		if oStr := r.URL.Query().Get("offset"); oStr != "" {
			if parsedO, parseErr := strconv.Atoi(oStr); parseErr == nil && parsedO >= 0 {
				offset = parsedO
			}
		}

		messages, err := database.ListMessagesWithAttachments(r.Context(), roomID, limit, offset)
		if err != nil {
			slog.Warn("failed to fetch messages for room", "room_id", roomID, "error", err)
			respondJSONError(w, http.StatusBadRequest, "Invalid room ID or query execution failed")
			return
		}

		_ = json.NewEncoder(w).Encode(messages)
	}
}

// GetRoomPresenceHandler returns list of online users (profiles) in a room.
func GetRoomPresenceHandler(database *db.DB, manager chat.PresenceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, ok := auth.GetUserID(r.Context())
		if !ok {
			respondJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		roomID := r.PathValue("roomID")
		if roomID == "" {
			respondJSONError(w, http.StatusBadRequest, "Room ID is required")
			return
		}

		canAccess, err := database.CanAccessRoom(r.Context(), roomID, userID)
		if err != nil || !canAccess {
			respondJSONError(w, http.StatusForbidden, "Access denied")
			return
		}

		onlineUsers := manager.OnlineUsers(roomID)
		_ = json.NewEncoder(w).Encode(onlineUsers)
	}
}
