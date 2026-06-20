package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pulsechat/backend/internal/db/postgres/sqlc"
	"github.com/pulsechat/backend/internal/domain"
)

type RoomRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewRoomRepository(pool *pgxpool.Pool) *RoomRepository {
	return &RoomRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *RoomRepository) Create(ctx context.Context, id, name, roomType string) (*domain.Room, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid room uuid: %w", err)
	}

	params := sqlc.CreateRoomParams{
		ID:   uid,
		Name: name,
		Type: roomType,
	}

	row, err := r.queries.CreateRoom(ctx, params)
	if err != nil {
		return nil, err
	}

	return &domain.Room{
		ID:        row.ID.String(),
		Name:      row.Name,
		Type:      row.Type,
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

func (r *RoomRepository) GetByID(ctx context.Context, id string) (*domain.Room, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid room uuid: %w", err)
	}

	row, err := r.queries.GetRoomByID(ctx, uid)
	if err != nil {
		return nil, err
	}

	return &domain.Room{
		ID:        row.ID.String(),
		Name:      row.Name,
		Type:      row.Type,
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

func (r *RoomRepository) ListForUser(ctx context.Context, userID string) ([]domain.RoomWithMeta, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user UUID: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT r.id, r.name, r.type, r.created_at,
		        CASE
		          WHEN r.type = 'direct' THEN COALESCE(ou.username, r.name)
		          ELSE r.name
		        END AS display_name,
		        CASE
		          WHEN r.type = 'direct' THEN ou.id::text
		          ELSE ''
		        END AS other_user_id,
		        CASE
		          WHEN r.type = 'direct' THEN COALESCE(ou.avatar_url, '')
		          ELSE ''
		        END AS other_user_avatar_url,
		        (SELECT COUNT(*)::int FROM room_members rm2 WHERE rm2.room_id = r.id) AS member_count
		 FROM rooms r
		 INNER JOIN room_members rm ON rm.room_id = r.id AND rm.user_id = $1
		 LEFT JOIN room_members om ON om.room_id = r.id AND om.user_id != $1 AND r.type = 'direct'
		 LEFT JOIN users ou ON ou.id = om.user_id
		 WHERE r.type IN ('direct', 'group', 'private')
		 ORDER BY COALESCE(
		   (SELECT MAX(m.created_at) FROM messages m WHERE m.room_id = r.id),
		   r.created_at
		 ) DESC`,
		uid)
	if err != nil {
		return nil, fmt.Errorf("failed to list rooms for user: %w", err)
	}
	defer rows.Close()

	var rooms []domain.RoomWithMeta
	for rows.Next() {
		var rm domain.RoomWithMeta
		var id uuid.UUID
		var otherUserID *string
		var otherUserAvatar *string
		if err := rows.Scan(&id, &rm.Name, &rm.Type, &rm.CreatedAt, &rm.DisplayName, &otherUserID, &otherUserAvatar, &rm.MemberCount); err != nil {
			return nil, fmt.Errorf("failed to scan room row: %w", err)
		}
		rm.ID = id.String()
		if otherUserID != nil {
			rm.OtherUserID = *otherUserID
		}
		if otherUserAvatar != nil {
			rm.OtherUserAvatarURL = *otherUserAvatar
		}
		if rm.Type == "private" {
			rm.Type = "group"
		}
		rooms = append(rooms, rm)
	}
	if rooms == nil {
		rooms = []domain.RoomWithMeta{}
	}
	return rooms, nil
}

func (r *RoomRepository) IsMember(ctx context.Context, roomID, userID string) (bool, error) {
	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return false, fmt.Errorf("invalid room UUID: %w", err)
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return false, fmt.Errorf("invalid user UUID: %w", err)
	}

	var exists bool
	err = r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM room_members WHERE room_id = $1 AND user_id = $2)`,
		roomUUID, userUUID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check room membership: %w", err)
	}
	return exists, nil
}

func (r *RoomRepository) GetMembers(ctx context.Context, roomID string) ([]domain.RoomMemberProfile, error) {
	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return nil, fmt.Errorf("invalid room UUID: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT u.id, u.username, u.email, u.avatar_url, rm.joined_at, rm.is_admin
		 FROM room_members rm
		 JOIN users u ON u.id = rm.user_id
		 WHERE rm.room_id = $1
		 ORDER BY rm.is_admin DESC, u.username ASC`,
		roomUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get room members: %w", err)
	}
	defer rows.Close()

	var members []domain.RoomMemberProfile
	for rows.Next() {
		var m domain.RoomMemberProfile
		var id uuid.UUID
		if err := rows.Scan(&id, &m.Username, &m.Email, &m.AvatarURL, &m.JoinedAt, &m.IsAdmin); err != nil {
			return nil, fmt.Errorf("failed to scan member row: %w", err)
		}
		m.ID = id.String()
		members = append(members, m)
	}
	if members == nil {
		members = []domain.RoomMemberProfile{}
	}
	return members, nil
}

func (r *RoomRepository) AddMember(ctx context.Context, roomID, userID string, isAdmin bool) error {
	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return fmt.Errorf("invalid room UUID: %w", err)
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user UUID: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO room_members (room_id, user_id, joined_at, is_admin)
		 VALUES ($1, $2, CURRENT_TIMESTAMP, $3)
		 ON CONFLICT (room_id, user_id) DO NOTHING`,
		roomUUID, userUUID, isAdmin)
	if err != nil {
		return fmt.Errorf("failed to add room member: %w", err)
	}
	return nil
}

func (r *RoomRepository) RemoveMember(ctx context.Context, roomID, userID string) error {
	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return fmt.Errorf("invalid room UUID: %w", err)
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user UUID: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`DELETE FROM room_members WHERE room_id = $1 AND user_id = $2`,
		roomUUID, userUUID)
	if err != nil {
		return fmt.Errorf("failed to remove room member: %w", err)
	}
	return nil
}

func (r *RoomRepository) IsAdmin(ctx context.Context, roomID, userID string) (bool, error) {
	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return false, fmt.Errorf("invalid room UUID: %w", err)
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return false, fmt.Errorf("invalid user UUID: %w", err)
	}

	var isAdmin bool
	err = r.pool.QueryRow(ctx,
		`SELECT is_admin FROM room_members WHERE room_id = $1 AND user_id = $2`,
		roomUUID, userUUID).Scan(&isAdmin)
	if err != nil {
		return false, fmt.Errorf("failed to check group admin: %w", err)
	}
	return isAdmin, nil
}

func (r *RoomRepository) CountAdmins(ctx context.Context, roomID string) (int, error) {
	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return 0, fmt.Errorf("invalid room UUID: %w", err)
	}

	var count int
	err = r.pool.QueryRow(ctx,
		`SELECT COUNT(*)::int FROM room_members WHERE room_id = $1 AND is_admin = true`,
		roomUUID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count group admins: %w", err)
	}
	return count, nil
}

func (r *RoomRepository) Delete(ctx context.Context, roomID string) error {
	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return fmt.Errorf("invalid room UUID: %w", err)
	}

	_, err = r.pool.Exec(ctx, `DELETE FROM rooms WHERE id = $1`, roomUUID)
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}
	return nil
}

func (r *RoomRepository) FindDirectRoom(ctx context.Context, user1ID, user2ID string) (*domain.RoomWithMeta, error) {
	u1, err := uuid.Parse(user1ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user UUID: %w", err)
	}
	u2, err := uuid.Parse(user2ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user UUID: %w", err)
	}

	row := r.pool.QueryRow(ctx,
		`SELECT r.id, r.name, r.type, r.created_at
		 FROM rooms r
		 JOIN room_members rm1 ON rm1.room_id = r.id AND rm1.user_id = $1
		 JOIN room_members rm2 ON rm2.room_id = r.id AND rm2.user_id = $2
		 WHERE r.type = 'direct'
		 LIMIT 1`,
		u1, u2)

	var rm domain.RoomWithMeta
	var id uuid.UUID
	if err := row.Scan(&id, &rm.Name, &rm.Type, &rm.CreatedAt); err != nil {
		return nil, err
	}
	rm.ID = id.String()
	return &rm, nil
}

func (r *RoomRepository) CreateDirectRoom(ctx context.Context, user1ID, user2ID, otherUsername, otherAvatarURL string) (*domain.RoomWithMeta, error) {
	u1, err := uuid.Parse(user1ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user UUID: %w", err)
	}
	u2, err := uuid.Parse(user2ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user UUID: %w", err)
	}

	name := directRoomName(user1ID, user2ID)
	roomID := uuid.New()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx,
		`INSERT INTO rooms (id, name, type, created_at) VALUES ($1, $2, 'direct', CURRENT_TIMESTAMP)`,
		roomID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to create direct room: %w", err)
	}

	for _, uid := range []uuid.UUID{u1, u2} {
		_, err = tx.Exec(ctx,
			`INSERT INTO room_members (room_id, user_id, joined_at) VALUES ($1, $2, CURRENT_TIMESTAMP)`,
			roomID, uid)
		if err != nil {
			return nil, fmt.Errorf("failed to add direct room member: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit direct room: %w", err)
	}

	return &domain.RoomWithMeta{
		ID:                 roomID.String(),
		Name:               name,
		Type:               "direct",
		DisplayName:        otherUsername,
		OtherUserID:        user2ID,
		OtherUserAvatarURL: otherAvatarURL,
		MemberCount:        2,
	}, nil
}

func directRoomName(user1ID, user2ID string) string {
	if user1ID < user2ID {
		return "dm:" + user1ID + ":" + user2ID
	}
	return "dm:" + user2ID + ":" + user1ID
}

func (r *RoomRepository) GetMembersForMentions(ctx context.Context, roomID string) (string, string, []domain.MemberBrief, error) {
	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return "", "", nil, fmt.Errorf("invalid room UUID: %w", err)
	}

	room, err := r.queries.GetRoomByID(ctx, roomUUID)
	if err != nil {
		return "", "", nil, fmt.Errorf("room not found: %w", err)
	}

	roomType := room.Type
	if roomType == "private" {
		roomType = "group"
	}
	roomName := room.Name

	profiles, err := r.GetMembers(ctx, roomID)
	if err != nil {
		return "", "", nil, err
	}

	members := make([]domain.MemberBrief, 0, len(profiles))
	for _, p := range profiles {
		members = append(members, domain.MemberBrief{
			UserID:   p.ID,
			Username: p.Username,
		})
	}
	return roomType, roomName, members, nil
}
