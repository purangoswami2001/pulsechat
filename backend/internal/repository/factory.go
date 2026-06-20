package repository

import (
	"fmt"

	"github.com/pulsechat/backend/internal/db"
	"github.com/pulsechat/backend/internal/repository/mysql"
	"github.com/pulsechat/backend/internal/repository/postgres"
)

type Repositories struct {
	User    UserRepository
	Room    RoomRepository
	Message MessageRepository
}

func NewRepositories(conn *db.Connection) (*Repositories, error) {
	switch conn.Driver {
	case "postgres":
		return &Repositories{
			User:    postgres.NewUserRepository(conn.PGPool),
			Room:    postgres.NewRoomRepository(conn.PGPool),
			Message: postgres.NewMessageRepository(conn.PGPool),
		}, nil
	case "mysql":
		return &Repositories{
			User:    mysql.NewUserRepository(),
			Room:    mysql.NewRoomRepository(),
			Message: mysql.NewMessageRepository(),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported repository driver: %s", conn.Driver)
	}
}
