package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/pulsechat/backend/internal/auth"
	"github.com/pulsechat/backend/internal/chat"
	"github.com/pulsechat/backend/internal/config"
	"github.com/pulsechat/backend/internal/db"
	"github.com/pulsechat/backend/internal/handler"
	"github.com/pulsechat/backend/internal/pubsub"
)

type membersStoreAdapter struct {
	database *db.DB
}

func (a *membersStoreAdapter) GetRoomForMentions(ctx context.Context, roomID string) (string, string, []chat.MemberBrief, error) {
	roomType, roomName, members, err := a.database.GetRoomMembersForMentions(ctx, roomID)
	if err != nil {
		return "", "", nil, err
	}

	out := make([]chat.MemberBrief, len(members))
	for i, m := range members {
		out[i] = chat.MemberBrief{UserID: m.UserID, Username: m.Username}
	}
	return roomType, roomName, out, nil
}

func main() {
	// Initialize logger (Text format for development, JSON for production)
	var logHandler slog.Handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	if os.Getenv("ENV") == "production" {
		logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	slog.Info("starting PulseChat server initialization")

	// Load Configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Initialize Database Pool & Run Auto-Migrations
	database, err := db.Connect(context.Background(), cfg.DBDSN)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// Ensure uploads directory exists
	uploadDir := filepath.Join(".", "uploads")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		slog.Error("failed to create uploads directory", "error", err)
		os.Exit(1)
	}
	slog.Info("uploads directory ready", "path", uploadDir)

	// Initialize Store Adapter & Presence Manager
	messageStore := chat.NewSQLCMessageStore(database)
	presenceManager := chat.NewInMemoryPresence()

	// Initialize PubSub Driver
	ps, err := pubsub.New(cfg.PubSubDriver, cfg.RedisURL)
	if err != nil {
		slog.Error("failed to initialize PubSub driver", "driver", cfg.PubSubDriver, "error", err)
		os.Exit(1)
	}
	defer ps.Close()
	slog.Info("PubSub driver initialized", "driver", cfg.PubSubDriver)

	// Initialize and Run the WebSocket Hub
	hub := chat.NewHub(messageStore, database, &membersStoreAdapter{database: database}, presenceManager, ps)
	hubCtx, cancelHub := context.WithCancel(context.Background())
	defer cancelHub()
	go hub.Run(hubCtx)

	// Setup Router
	mux := http.NewServeMux()

	// Instantiate Auth Middleware Wrapper for standard REST routes
	authMiddleware := auth.AuthMiddleware(cfg.JWTSecret)

	// Public Routes
	mux.HandleFunc("GET /health", handler.HealthHandler())
	mux.HandleFunc("POST /auth/register", handler.RegisterHandler(database, cfg))
	mux.HandleFunc("POST /auth/login", handler.LoginHandler(database, cfg))
	
	// WebSocket Endpoint (Validates auth token internally via query parameters)
	mux.HandleFunc("GET /ws", handler.WSHandler(hub, database, cfg.JWTSecret))

	// Static file server for uploads (avatars, attachments)
	fileServer := http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadDir)))
	mux.Handle("GET /uploads/", fileServer)

	// Protected REST Routes — Rooms
	mux.Handle("POST /rooms", authMiddleware(handler.CreateRoomHandler(database, hub)))
	mux.Handle("GET /rooms", authMiddleware(handler.ListRoomsHandler(database)))
	mux.Handle("POST /rooms/direct", authMiddleware(handler.CreateDirectRoomHandler(database)))
	mux.Handle("GET /rooms/{roomID}/messages", authMiddleware(handler.GetRoomMessagesHandler(database)))
	mux.Handle("GET /rooms/{roomID}/presence", authMiddleware(handler.GetRoomPresenceHandler(database, presenceManager)))
	mux.Handle("GET /rooms/{roomID}/members", authMiddleware(handler.GetRoomMembersHandler(database)))
	mux.Handle("POST /rooms/{roomID}/members", authMiddleware(handler.AddRoomMemberHandler(database, hub)))
	mux.Handle("DELETE /rooms/{roomID}/members/{userID}", authMiddleware(handler.RemoveRoomMemberHandler(database)))
	mux.Handle("DELETE /rooms/{roomID}", authMiddleware(handler.DeleteGroupHandler(database)))
	mux.Handle("POST /rooms/{roomID}/delete", authMiddleware(handler.DeleteGroupHandler(database)))

	// Protected REST Routes — Users
	mux.Handle("GET /users/search", authMiddleware(handler.SearchUsersHandler(database)))

	// Protected REST Routes — Profile
	mux.Handle("GET /auth/profile", authMiddleware(handler.GetProfileHandler(database)))
	mux.Handle("PUT /auth/profile", authMiddleware(handler.UpdateProfileHandler(database)))
	mux.Handle("POST /auth/avatar", authMiddleware(handler.UploadAvatarHandler(database, uploadDir)))
	mux.Handle("DELETE /auth/avatar", authMiddleware(handler.RemoveAvatarHandler(database, uploadDir)))

	// Protected REST Routes — File Upload
	mux.Handle("POST /upload", authMiddleware(handler.UploadHandler(uploadDir)))

	// Custom CORS Middleware
	corsMux := enableCORS(mux)

	// Configure Server
	serverAddr := fmt.Sprintf(":%s", cfg.Port)
	server := &http.Server{
		Addr:         serverAddr,
		Handler:      corsMux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful Shutdown Channel
	shutdownError := make(chan error, 1)

	go func() {
		slog.Info("HTTP server listening", "addr", serverAddr, "env", cfg.Env)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			shutdownError <- err
		}
	}()

	// Wait for termination signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		slog.Info("received shutdown signal", "signal", sig.String())
	case err := <-shutdownError:
		slog.Error("server error, forcing shutdown", "error", err)
		os.Exit(1)
	}

	// Cancel Hub context to stop execution loops and release connections
	cancelHub()

	// Shutdown with context timeout
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	slog.Info("shutting down HTTP server gracefully...")
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped gracefully")
}

// enableCORS is a middleware that handles CORS requests, allowing frontend connections.
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set headers
		w.Header().Set("Access-Control-Allow-Origin", "*") // Adjust to specific origin in production if needed
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
