package postgres

// Store is a placeholder/wrapper if database-wide operations are needed.
type Store struct {
	User    *UserRepository
	Room    *RoomRepository
	Message *MessageRepository
}
