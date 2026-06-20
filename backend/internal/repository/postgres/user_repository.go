package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pulsechat/backend/internal/db/postgres/sqlc"
	"github.com/pulsechat/backend/internal/domain"
)

type UserRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *UserRepository) Create(ctx context.Context, id, username, email, passwordHash string) (*domain.User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user uuid: %w", err)
	}

	params := sqlc.CreateUserParams{
		ID:           uid,
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
	}

	row, err := r.queries.CreateUser(ctx, params)
	if err != nil {
		return nil, err
	}

	return &domain.User{
		ID:        row.ID.String(),
		Username:  row.Username,
		Email:     row.Email,
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user uuid: %w", err)
	}

	row, err := r.queries.GetUserByID(ctx, uid)
	if err != nil {
		return nil, err
	}

	return &domain.User{
		ID:        row.ID.String(),
		Username:  row.Username,
		Email:     row.Email,
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return &domain.User{
		ID:           row.ID.String(),
		Username:     row.Username,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		CreatedAt:    row.CreatedAt.Time,
	}, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, username, email, avatar_url, created_at FROM users WHERE username = $1`, username)

	var u domain.User
	var uid uuid.UUID
	if err := row.Scan(&uid, &u.Username, &u.Email, &u.AvatarURL, &u.CreatedAt); err != nil {
		return nil, fmt.Errorf("user not found by username: %w", err)
	}
	u.ID = uid.String()
	return &u, nil
}

func (r *UserRepository) GetProfile(ctx context.Context, id string) (*domain.User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user UUID: %w", err)
	}

	row := r.pool.QueryRow(ctx,
		`SELECT id, username, email, avatar_url, created_at FROM users WHERE id = $1`, uid)

	var u domain.User
	var uID uuid.UUID
	if err := row.Scan(&uID, &u.Username, &u.Email, &u.AvatarURL, &u.CreatedAt); err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}
	u.ID = uID.String()
	return &u, nil
}

func (r *UserRepository) UpdateProfile(ctx context.Context, id, username, email string) (*domain.User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user UUID: %w", err)
	}

	row := r.pool.QueryRow(ctx,
		`UPDATE users SET username = $2, email = $3 WHERE id = $1
		 RETURNING id, username, email, avatar_url, created_at`,
		uid, username, email)

	var u domain.User
	var uID uuid.UUID
	if err := row.Scan(&uID, &u.Username, &u.Email, &u.AvatarURL, &u.CreatedAt); err != nil {
		return nil, fmt.Errorf("failed to update user profile: %w", err)
	}
	u.ID = uID.String()
	return &u, nil
}

func (r *UserRepository) UpdateAvatar(ctx context.Context, id, avatarURL string) (*domain.User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user UUID: %w", err)
	}

	row := r.pool.QueryRow(ctx,
		`UPDATE users SET avatar_url = $2 WHERE id = $1
		 RETURNING id, username, email, avatar_url, created_at`,
		uid, avatarURL)

	var u domain.User
	var uID uuid.UUID
	if err := row.Scan(&uID, &u.Username, &u.Email, &u.AvatarURL, &u.CreatedAt); err != nil {
		return nil, fmt.Errorf("failed to update avatar: %w", err)
	}
	u.ID = uID.String()
	return &u, nil
}

func (r *UserRepository) Search(ctx context.Context, query, excludeUserID string, limit int) ([]domain.User, error) {
	if limit <= 0 || limit > 20 {
		limit = 10
	}
	pattern := "%" + query + "%"

	excludeUUID, err := uuid.Parse(excludeUserID)
	if err != nil {
		return nil, fmt.Errorf("invalid exclude user UUID: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, username, email, avatar_url, created_at
		 FROM users
		 WHERE id != $1
		   AND (username ILIKE $2 OR email ILIKE $2)
		 ORDER BY username ASC
		 LIMIT $3`,
		excludeUUID, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		var uID uuid.UUID
		if err := rows.Scan(&uID, &u.Username, &u.Email, &u.AvatarURL, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		u.ID = uID.String()
		users = append(users, u)
	}
	if users == nil {
		users = []domain.User{}
	}
	return users, nil
}

func (r *UserRepository) GetContacts(ctx context.Context, userID string) ([]string, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user UUID: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT user_id 
		FROM room_members 
		WHERE room_id IN (
			SELECT room_id 
			FROM room_members 
			WHERE user_id = $1
		) AND user_id != $1
	`, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to query contacts: %w", err)
	}
	defer rows.Close()

	var contacts []string
	for rows.Next() {
		var contactUUID uuid.UUID
		if err := rows.Scan(&contactUUID); err != nil {
			return nil, fmt.Errorf("failed to scan contact: %w", err)
		}
		contacts = append(contacts, contactUUID.String())
	}
	if contacts == nil {
		contacts = []string{}
	}
	return contacts, nil
}
