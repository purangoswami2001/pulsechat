package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pulsechat/backend/internal/config"
	"github.com/pulsechat/backend/internal/db"
	"github.com/pulsechat/backend/internal/pubsub"
	"github.com/pulsechat/backend/internal/repository"
	"github.com/pulsechat/backend/internal/service"
	httpTransport "github.com/pulsechat/backend/internal/transport/http"
	"github.com/pulsechat/backend/internal/transport/http/handler"
	"github.com/pulsechat/backend/internal/transport/websocket"
	"github.com/pulsechat/backend/internal/storage"
)

type Dependencies struct {
	Config          *config.Config
	DBConnection    *db.Connection
	PubSub          pubsub.PubSub
	Repositories    *repository.Repositories
	AuthService     *service.AuthService
	UserService     *service.UserService
	RoomService     *service.RoomService
	MessageService  *service.MessageService
	UploadService   *service.UploadService
	PresenceService *service.PresenceService
	WSHub           *websocket.Hub
	HTTPHandler     http.Handler
}

func NewDependencies(ctx context.Context) (*Dependencies, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	conn, err := db.Connect(ctx, cfg.DBDriver, cfg.DBDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	ps, err := pubsub.New(cfg.PubSubDriver, cfg.RedisURL)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to load pubsub: %w", err)
	}

	repos, err := repository.NewRepositories(conn)
	if err != nil {
		conn.Close()
		_ = ps.Close()
		return nil, fmt.Errorf("failed to load repositories: %w", err)
	}

	// Instantiate Services
	authSvc := service.NewAuthService(repos.User, cfg.JWTSecret, cfg.JWTExpirationHours)
	userSvc := service.NewUserService(repos.User)
	roomSvc := service.NewRoomService(repos.Room, repos.User)
	msgSvc := service.NewMessageService(repos.Message)

	localStore := storage.NewLocalStorage("./uploads")
	uploadSvc := service.NewUploadService(localStore)
	presenceSvc := service.NewPresenceService()

	// Instantiate WS Hub (which serves as our Notifier)
	hub := websocket.NewHub(userSvc, roomSvc, msgSvc, presenceSvc, ps)

	// Instantiate Handlers
	authHandler := handler.NewAuthHandler(authSvc)
	userHandler := handler.NewUserHandler(userSvc, uploadSvc)
	roomHandler := handler.NewRoomHandler(roomSvc, msgSvc, presenceSvc, hub)
	uploadHandler := handler.NewUploadHandler(uploadSvc)

	wsHandlerFunc := websocket.WSHandler(hub, roomSvc, userSvc, cfg.JWTSecret)

	router := httpTransport.NewRouter(httpTransport.RouterConfig{
		AuthHandler:   authHandler,
		UserHandler:   userHandler,
		RoomHandler:   roomHandler,
		UploadHandler: uploadHandler,
		WSHandler:     wsHandlerFunc,
		JWTSecret:     cfg.JWTSecret,
		UploadsDir:    "./uploads",
	})

	return &Dependencies{
		Config:          cfg,
		DBConnection:    conn,
		PubSub:          ps,
		Repositories:    repos,
		AuthService:     authSvc,
		UserService:     userSvc,
		RoomService:     roomSvc,
		MessageService:  msgSvc,
		UploadService:   uploadSvc,
		PresenceService: presenceSvc,
		WSHub:           hub,
		HTTPHandler:     router,
	}, nil
}

func (d *Dependencies) Close() {
	if d.DBConnection != nil {
		d.DBConnection.Close()
	}
	if d.PubSub != nil {
		_ = d.PubSub.Close()
	}
}
