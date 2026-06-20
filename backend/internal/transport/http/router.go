package http

import (
	"net/http"

	"github.com/pulsechat/backend/internal/transport/http/handler"
	"github.com/pulsechat/backend/internal/transport/http/middleware"
)

type RouterConfig struct {
	AuthHandler   *handler.AuthHandler
	UserHandler   *handler.UserHandler
	RoomHandler   *handler.RoomHandler
	UploadHandler *handler.UploadHandler
	WSHandler     http.HandlerFunc
	JWTSecret     string
	UploadsDir    string
}

func NewRouter(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()

	authMiddleware := middleware.AuthMiddleware(cfg.JWTSecret)

	// Public Routes
	mux.HandleFunc("GET /health", handler.HealthHandler())
	mux.HandleFunc("POST /auth/register", cfg.AuthHandler.Register())
	mux.HandleFunc("POST /auth/login", cfg.AuthHandler.Login())

	// WebSocket Route
	mux.HandleFunc("GET /ws", cfg.WSHandler)

	// File Upload Static Server
	fileServer := http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.UploadsDir)))
	mux.Handle("GET /uploads/", fileServer)

	// Protected REST Routes — Rooms
	mux.Handle("POST /rooms", authMiddleware(cfg.RoomHandler.CreateRoom()))
	mux.Handle("GET /rooms", authMiddleware(cfg.RoomHandler.ListRooms()))
	mux.Handle("POST /rooms/direct", authMiddleware(cfg.RoomHandler.CreateDirectRoom()))
	mux.Handle("GET /rooms/{roomID}/messages", authMiddleware(cfg.RoomHandler.GetRoomMessages()))
	mux.Handle("GET /rooms/{roomID}/presence", authMiddleware(cfg.RoomHandler.GetRoomPresence()))
	mux.Handle("GET /rooms/{roomID}/members", authMiddleware(cfg.RoomHandler.GetRoomMembers()))
	mux.Handle("POST /rooms/{roomID}/members", authMiddleware(cfg.RoomHandler.AddRoomMember()))
	mux.Handle("DELETE /rooms/{roomID}/members/{userID}", authMiddleware(cfg.RoomHandler.RemoveRoomMember()))
	mux.Handle("DELETE /rooms/{roomID}", authMiddleware(cfg.RoomHandler.DeleteGroup()))
	mux.Handle("POST /rooms/{roomID}/delete", authMiddleware(cfg.RoomHandler.DeleteGroup()))

	// Protected REST Routes — Users
	mux.Handle("GET /users/search", authMiddleware(cfg.UserHandler.SearchUsers()))

	// Protected REST Routes — Profile
	mux.Handle("GET /auth/profile", authMiddleware(cfg.UserHandler.GetProfile()))
	mux.Handle("PUT /auth/profile", authMiddleware(cfg.UserHandler.UpdateProfile())) // Support PUT updates
	mux.Handle("POST /auth/avatar", authMiddleware(cfg.UserHandler.UploadAvatar()))
	mux.Handle("DELETE /auth/avatar", authMiddleware(cfg.UserHandler.RemoveAvatar()))

	// Protected REST Routes — Upload
	mux.Handle("POST /upload", authMiddleware(cfg.UploadHandler.Upload()))

	// Wrap in Global Middlewares (CORS & Recovery)
	var handler http.Handler = mux
	handler = middleware.EnableCORS(handler)
	handler = middleware.Recovery(handler)

	return handler
}
