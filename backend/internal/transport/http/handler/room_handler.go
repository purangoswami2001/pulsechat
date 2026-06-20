package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/pulsechat/backend/internal/auth"
	"github.com/pulsechat/backend/internal/service"
)

// Notifier abstracts websocket notifications.
type Notifier interface {
	NotifyGroupInvite(userID string, payload interface{})
}

type RoomHandler struct {
	roomService *service.RoomService
	msgService  *service.MessageService
	presenceSvc *service.PresenceService
	notifier    Notifier
}

func NewRoomHandler(roomSvc *service.RoomService, msgSvc *service.MessageService, presenceSvc *service.PresenceService, notifier Notifier) *RoomHandler {
	return &RoomHandler{
		roomService: roomSvc,
		msgService:  msgSvc,
		presenceSvc: presenceSvc,
		notifier:    notifier,
	}
}

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

type GroupInviteNotification struct {
	GroupID     string `json:"group_id"`
	GroupName   string `json:"group_name"`
	InviterID   string `json:"inviter_id"`
	InviterName string `json:"inviter_name"`
}

func (h *RoomHandler) CreateRoom() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		creatorIDStr, ok := auth.GetUserID(r.Context())
		if !ok {
			respondJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		creatorName, _ := auth.GetUsername(r.Context())

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

		if req.Type == "private" {
			req.Type = "group"
		}

		room, err := h.roomService.CreateRoom(r.Context(), req.Name, req.Type, creatorIDStr, req.MemberIDs)
		if err != nil {
			respondJSONError(w, http.StatusInternalServerError, "Failed to create group")
			return
		}

		for _, memberID := range req.MemberIDs {
			if memberID != "" && memberID != creatorIDStr {
				h.notifier.NotifyGroupInvite(memberID, GroupInviteNotification{
					GroupID:     room.ID,
					GroupName:   room.Name,
					InviterID:   creatorIDStr,
					InviterName: creatorName,
				})
			}
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(room)
	}
}

func (h *RoomHandler) ListRooms() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, ok := auth.GetUserID(r.Context())
		if !ok {
			respondJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		rooms, err := h.roomService.ListRoomsForUser(r.Context(), userID)
		if err != nil {
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		_ = json.NewEncoder(w).Encode(rooms)
	}
}

func (h *RoomHandler) CreateDirectRoom() http.HandlerFunc {
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

		room, err := h.roomService.GetOrCreateDirectRoom(r.Context(), callerID, req.UserID)
		if err != nil {
			respondJSONError(w, http.StatusInternalServerError, "Failed to create direct room")
			return
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(room)
	}
}

func (h *RoomHandler) GetRoomMembers() http.HandlerFunc {
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

		canAccess, err := h.roomService.IsRoomMember(r.Context(), roomID, userID)
		if err != nil || !canAccess {
			respondJSONError(w, http.StatusForbidden, "Access denied")
			return
		}

		members, err := h.roomService.GetRoomMembers(r.Context(), roomID)
		if err != nil {
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		_ = json.NewEncoder(w).Encode(members)
	}
}

func (h *RoomHandler) AddRoomMember() http.HandlerFunc {
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

		room, err := h.roomService.GetByID(r.Context(), roomID)
		if err != nil {
			respondJSONError(w, http.StatusNotFound, "Room not found")
			return
		}

		if room.Type != "group" && room.Type != "private" {
			respondJSONError(w, http.StatusBadRequest, "Members can only be added to group chats")
			return
		}

		isAdmin, err := h.roomService.IsGroupAdmin(r.Context(), roomID, callerID)
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
		if targetUserID == "" {
			respondJSONError(w, http.StatusBadRequest, "user_id is required")
			return
		}

		if err := h.roomService.AddRoomMember(r.Context(), roomID, targetUserID, false); err != nil {
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		h.notifier.NotifyGroupInvite(targetUserID, GroupInviteNotification{
			GroupID:     roomID,
			GroupName:   room.Name,
			InviterID:   callerID,
			InviterName: callerName,
		})

		members, err := h.roomService.GetRoomMembers(r.Context(), roomID)
		if err != nil {
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(members)
	}
}

func (h *RoomHandler) DeleteGroup() http.HandlerFunc {
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

		room, err := h.roomService.GetByID(r.Context(), roomID)
		if err != nil {
			respondJSONError(w, http.StatusNotFound, "Group not found")
			return
		}

		if room.Type != "group" && room.Type != "private" && room.Type != "direct" {
			respondJSONError(w, http.StatusBadRequest, "Only groups and direct chats can be deleted")
			return
		}

		if room.Type == "group" || room.Type == "private" {
			isAdmin, err := h.roomService.IsGroupAdmin(r.Context(), roomID, callerID)
			if err != nil || !isAdmin {
				respondJSONError(w, http.StatusForbidden, "Only group admins can delete the group")
				return
			}
		} else if room.Type == "direct" {
			canAccess, err := h.roomService.IsRoomMember(r.Context(), roomID, callerID)
			if err != nil || !canAccess {
				respondJSONError(w, http.StatusForbidden, "Only participants can delete this direct chat")
				return
			}
		}

		if err := h.roomService.DeleteGroup(r.Context(), roomID); err != nil {
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func (h *RoomHandler) RemoveRoomMember() http.HandlerFunc {
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

		room, err := h.roomService.GetByID(r.Context(), roomID)
		if err != nil {
			respondJSONError(w, http.StatusNotFound, "Group not found")
			return
		}

		if room.Type != "group" && room.Type != "private" {
			respondJSONError(w, http.StatusBadRequest, "Members can only be removed from group chats")
			return
		}

		isAdmin, err := h.roomService.IsGroupAdmin(r.Context(), roomID, callerID)
		if err != nil || !isAdmin {
			respondJSONError(w, http.StatusForbidden, "Only group admins can remove members")
			return
		}

		if targetUserID == callerID {
			respondJSONError(w, http.StatusBadRequest, "Admins cannot remove themselves")
			return
		}

		targetIsAdmin, err := h.roomService.IsGroupAdmin(r.Context(), roomID, targetUserID)
		if err == nil && targetIsAdmin {
			adminCount, countErr := h.roomService.CountGroupAdmins(r.Context(), roomID)
			if countErr == nil && adminCount <= 1 {
				respondJSONError(w, http.StatusBadRequest, "Cannot remove the only group admin")
				return
			}
		}

		if err := h.roomService.RemoveRoomMember(r.Context(), roomID, targetUserID); err != nil {
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		members, err := h.roomService.GetRoomMembers(r.Context(), roomID)
		if err != nil {
			respondJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		_ = json.NewEncoder(w).Encode(members)
	}
}

func (h *RoomHandler) GetRoomMessages() http.HandlerFunc {
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

		canAccess, err := h.roomService.IsRoomMember(r.Context(), roomID, userID)
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

		messages, err := h.msgService.ListMessages(r.Context(), roomID, limit, offset)
		if err != nil {
			respondJSONError(w, http.StatusBadRequest, "Invalid room ID or query execution failed")
			return
		}

		_ = json.NewEncoder(w).Encode(messages)
	}
}

func (h *RoomHandler) GetRoomPresence() http.HandlerFunc {
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

		canAccess, err := h.roomService.IsRoomMember(r.Context(), roomID, userID)
		if err != nil || !canAccess {
			respondJSONError(w, http.StatusForbidden, "Access denied")
			return
		}

		onlineUsers := h.presenceSvc.OnlineUsers(roomID)
		_ = json.NewEncoder(w).Encode(onlineUsers)
	}
}
