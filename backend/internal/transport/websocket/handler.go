package websocket

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/pulsechat/backend/internal/auth"
	"github.com/pulsechat/backend/internal/service"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func WSHandler(hub *Hub, roomSvc *service.RoomService, userSvc *service.UserService, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomID := r.URL.Query().Get("room_id")
		tokenStr := r.URL.Query().Get("token")
		notifyOnly := r.URL.Query().Get("notify") == "1"

		if tokenStr == "" {
			http.Error(w, "token query parameter is required", http.StatusBadRequest)
			return
		}

		if !notifyOnly && roomID == "" {
			http.Error(w, "room_id query parameter is required", http.StatusBadRequest)
			return
		}

		claims, err := auth.ValidateToken(tokenStr, jwtSecret)
		if err != nil {
			slog.Warn("WebSocket upgrade rejected: invalid auth token", "error", err)
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}

		if !notifyOnly {
			canAccess, accessErr := roomSvc.IsRoomMember(r.Context(), roomID, claims.UserID)
			if accessErr != nil || !canAccess {
				slog.Warn("WebSocket upgrade rejected: access denied", "room_id", roomID, "user_id", claims.UserID)
				http.Error(w, "Forbidden: you do not have access to this room", http.StatusForbidden)
				return
			}
		}

		avatarURL := ""
		currentUser, err := userSvc.GetProfile(r.Context(), claims.UserID)
		if err == nil {
			avatarURL = currentUser.AvatarURL
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("failed to upgrade connection to WebSocket", "error", err)
			return
		}

		var client *Client
		if notifyOnly {
			client = NewNotifyClient(hub, conn, claims.UserID, claims.Username, avatarURL)
		} else {
			client = NewClient(hub, conn, roomID, claims.UserID, claims.Username, avatarURL)
		}

		hub.Register <- client

		go client.WritePump()
		client.ReadPump()
	}
}
