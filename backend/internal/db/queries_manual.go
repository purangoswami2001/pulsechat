package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// UserProfile holds user data including avatar for profile endpoints.
type UserProfile struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
}

// GetUserProfile fetches a user's full profile including avatar_url.
func (db *DB) GetUserProfile(ctx context.Context, userID string) (*UserProfile, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user UUID: %w", err)
	}

	row := db.Pool.QueryRow(ctx,
		`SELECT id, username, email, avatar_url, created_at FROM users WHERE id = $1`, uid)

	var p UserProfile
	var id uuid.UUID
	var createdAt time.Time
	if err := row.Scan(&id, &p.Username, &p.Email, &p.AvatarURL, &createdAt); err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}
	p.ID = id.String()
	p.CreatedAt = createdAt
	return &p, nil
}

// UpdateUserProfile updates a user's username and email.
func (db *DB) UpdateUserProfile(ctx context.Context, userID, username, email string) (*UserProfile, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user UUID: %w", err)
	}

	row := db.Pool.QueryRow(ctx,
		`UPDATE users SET username = $2, email = $3 WHERE id = $1
		 RETURNING id, username, email, avatar_url, created_at`,
		uid, username, email)

	var p UserProfile
	var id uuid.UUID
	var createdAt time.Time
	if err := row.Scan(&id, &p.Username, &p.Email, &p.AvatarURL, &createdAt); err != nil {
		return nil, fmt.Errorf("failed to update user profile: %w", err)
	}
	p.ID = id.String()
	p.CreatedAt = createdAt
	return &p, nil
}

// UpdateUserAvatar sets or clears a user's avatar URL.
func (db *DB) UpdateUserAvatar(ctx context.Context, userID, avatarURL string) (*UserProfile, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user UUID: %w", err)
	}

	row := db.Pool.QueryRow(ctx,
		`UPDATE users SET avatar_url = $2 WHERE id = $1
		 RETURNING id, username, email, avatar_url, created_at`,
		uid, avatarURL)

	var p UserProfile
	var id uuid.UUID
	var createdAt time.Time
	if err := row.Scan(&id, &p.Username, &p.Email, &p.AvatarURL, &createdAt); err != nil {
		return nil, fmt.Errorf("failed to update avatar: %w", err)
	}
	p.ID = id.String()
	p.CreatedAt = createdAt
	return &p, nil
}

// CreateMessageWithAttachment inserts a message with optional attachment fields.
func (db *DB) CreateMessageWithAttachment(ctx context.Context, id, roomID, senderID uuid.UUID, content, attachmentURL, attachmentType string) error {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO messages (id, room_id, sender_id, content, attachment_url, attachment_type, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP)`,
		id, roomID, senderID, content, attachmentURL, attachmentType)
	if err != nil {
		return fmt.Errorf("failed to create message with attachment: %w", err)
	}
	return nil
}

// MessageWithAttachment holds message data including attachment info.
type MessageWithAttachment struct {
	ID              string    `json:"id"`
	RoomID          string    `json:"room_id"`
	SenderID        string    `json:"sender_id"`
	SenderName      string    `json:"sender_name"`
	SenderAvatarURL string    `json:"sender_avatar_url"`
	Content         string    `json:"content"`
	AttachmentURL   string    `json:"attachment_url"`
	AttachmentType  string    `json:"attachment_type"`
	CreatedAt       time.Time `json:"created_at"`
}

// ListMessagesWithAttachments fetches messages including attachment fields.
func (db *DB) ListMessagesWithAttachments(ctx context.Context, roomID string, limit, offset int) ([]MessageWithAttachment, error) {
	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return nil, fmt.Errorf("invalid room UUID: %w", err)
	}

	rows, err := db.Pool.Query(ctx,
		`SELECT m.id, m.room_id, m.sender_id, u.username AS sender_name, u.avatar_url AS sender_avatar_url,
		        m.content, m.attachment_url, m.attachment_type, m.created_at
		 FROM messages m
		 JOIN users u ON m.sender_id = u.id
		 WHERE m.room_id = $1
		 ORDER BY m.created_at ASC
		 LIMIT $2 OFFSET $3`,
		roomUUID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}
	defer rows.Close()

	var messages []MessageWithAttachment
	for rows.Next() {
		var m MessageWithAttachment
		var id, rID, sID uuid.UUID
		var createdAt time.Time
		if err := rows.Scan(&id, &rID, &sID, &m.SenderName, &m.SenderAvatarURL, &m.Content,
			&m.AttachmentURL, &m.AttachmentType, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}
		m.ID = id.String()
		m.RoomID = rID.String()
		m.SenderID = sID.String()
		m.CreatedAt = createdAt
		messages = append(messages, m)
	}

	if messages == nil {
		messages = []MessageWithAttachment{}
	}
	return messages, nil
}

// GetUserAvatarURL fetches just the avatar URL for a given user ID.
func (db *DB) GetUserAvatarURL(ctx context.Context, userID string) (string, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", fmt.Errorf("invalid user UUID: %w", err)
	}

	var avatarURL string
	err = db.Pool.QueryRow(ctx, `SELECT avatar_url FROM users WHERE id = $1`, uid).Scan(&avatarURL)
	if err != nil {
		return "", fmt.Errorf("failed to get avatar URL: %w", err)
	}
	return avatarURL, nil
}

// UserSearchResult is a lightweight user record for search/autocomplete.
type UserSearchResult struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// SearchUsers finds users by username or email prefix, excluding the caller.
func (db *DB) SearchUsers(ctx context.Context, query, excludeUserID string, limit int) ([]UserSearchResult, error) {
	if limit <= 0 || limit > 20 {
		limit = 10
	}
	pattern := "%" + query + "%"

	excludeUUID, err := uuid.Parse(excludeUserID)
	if err != nil {
		return nil, fmt.Errorf("invalid exclude user UUID: %w", err)
	}

	rows, err := db.Pool.Query(ctx,
		`SELECT id, username, email, avatar_url
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

	var users []UserSearchResult
	for rows.Next() {
		var u UserSearchResult
		var id uuid.UUID
		if err := rows.Scan(&id, &u.Username, &u.Email, &u.AvatarURL); err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		u.ID = id.String()
		users = append(users, u)
	}
	if users == nil {
		users = []UserSearchResult{}
	}
	return users, nil
}

// RoomWithMeta extends a room with display info for the requesting user.
type RoomWithMeta struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	CreatedAt   time.Time `json:"created_at"`
	DisplayName         string    `json:"display_name,omitempty"`
	OtherUserID         string    `json:"other_user_id,omitempty"`
	OtherUserAvatarURL  string    `json:"other_user_avatar_url,omitempty"`
	MemberCount         int       `json:"member_count,omitempty"`
}

// ListRoomsForUser returns direct and group chats the user belongs to.
func (db *DB) ListRoomsForUser(ctx context.Context, userID string) ([]RoomWithMeta, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user UUID: %w", err)
	}

	rows, err := db.Pool.Query(ctx,
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

	var rooms []RoomWithMeta
	for rows.Next() {
		var r RoomWithMeta
		var id uuid.UUID
		var otherUserID *string
		var otherUserAvatar *string
		if err := rows.Scan(&id, &r.Name, &r.Type, &r.CreatedAt, &r.DisplayName, &otherUserID, &otherUserAvatar, &r.MemberCount); err != nil {
			return nil, fmt.Errorf("failed to scan room row: %w", err)
		}
		r.ID = id.String()
		if otherUserID != nil {
			r.OtherUserID = *otherUserID
		}
		if otherUserAvatar != nil {
			r.OtherUserAvatarURL = *otherUserAvatar
		}
		if r.Type == "private" {
			r.Type = "group"
		}
		rooms = append(rooms, r)
	}
	if rooms == nil {
		rooms = []RoomWithMeta{}
	}
	return rooms, nil
}

// IsRoomMember checks whether a user belongs to a room.
func (db *DB) IsRoomMember(ctx context.Context, roomID, userID string) (bool, error) {
	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return false, fmt.Errorf("invalid room UUID: %w", err)
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return false, fmt.Errorf("invalid user UUID: %w", err)
	}

	var exists bool
	err = db.Pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM room_members WHERE room_id = $1 AND user_id = $2)`,
		roomUUID, userUUID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check room membership: %w", err)
	}
	return exists, nil
}

// CanAccessRoom returns true if the user is a member of the chat.
func (db *DB) CanAccessRoom(ctx context.Context, roomID, userID string) (bool, error) {
	return db.IsRoomMember(ctx, roomID, userID)
}

// RoomMemberProfile is a room member with avatar for UI display.
type RoomMemberProfile struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	AvatarURL string    `json:"avatar_url"`
	JoinedAt  time.Time `json:"joined_at"`
	IsAdmin   bool      `json:"is_admin"`
}

// GetRoomMembersWithAvatar lists persisted members of a room.
func (db *DB) GetRoomMembersWithAvatar(ctx context.Context, roomID string) ([]RoomMemberProfile, error) {
	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return nil, fmt.Errorf("invalid room UUID: %w", err)
	}

	rows, err := db.Pool.Query(ctx,
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

	var members []RoomMemberProfile
	for rows.Next() {
		var m RoomMemberProfile
		var id uuid.UUID
		if err := rows.Scan(&id, &m.Username, &m.Email, &m.AvatarURL, &m.JoinedAt, &m.IsAdmin); err != nil {
			return nil, fmt.Errorf("failed to scan member row: %w", err)
		}
		m.ID = id.String()
		members = append(members, m)
	}
	if members == nil {
		members = []RoomMemberProfile{}
	}
	return members, nil
}

// AddRoomMember adds a user to a room (idempotent).
func (db *DB) AddRoomMember(ctx context.Context, roomID, userID string, isAdmin bool) error {
	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return fmt.Errorf("invalid room UUID: %w", err)
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user UUID: %w", err)
	}

	_, err = db.Pool.Exec(ctx,
		`INSERT INTO room_members (room_id, user_id, joined_at, is_admin)
		 VALUES ($1, $2, CURRENT_TIMESTAMP, $3)
		 ON CONFLICT (room_id, user_id) DO NOTHING`,
		roomUUID, userUUID, isAdmin)
	if err != nil {
		return fmt.Errorf("failed to add room member: %w", err)
	}
	return nil
}

// IsGroupAdmin checks if a user is an admin of a group.
func (db *DB) IsGroupAdmin(ctx context.Context, roomID, userID string) (bool, error) {
	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return false, fmt.Errorf("invalid room UUID: %w", err)
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return false, fmt.Errorf("invalid user UUID: %w", err)
	}

	var isAdmin bool
	err = db.Pool.QueryRow(ctx,
		`SELECT is_admin FROM room_members WHERE room_id = $1 AND user_id = $2`,
		roomUUID, userUUID).Scan(&isAdmin)
	if err != nil {
		return false, fmt.Errorf("failed to check group admin: %w", err)
	}
	return isAdmin, nil
}

// DeleteGroup deletes a group and all related data (cascade).
func (db *DB) DeleteGroup(ctx context.Context, roomID string) error {
	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return fmt.Errorf("invalid room UUID: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `DELETE FROM rooms WHERE id = $1`, roomUUID)
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}
	return nil
}

// RemoveRoomMember removes a user from a group.
func (db *DB) RemoveRoomMember(ctx context.Context, roomID, userID string) error {
	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return fmt.Errorf("invalid room UUID: %w", err)
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user UUID: %w", err)
	}

	_, err = db.Pool.Exec(ctx,
		`DELETE FROM room_members WHERE room_id = $1 AND user_id = $2`,
		roomUUID, userUUID)
	if err != nil {
		return fmt.Errorf("failed to remove room member: %w", err)
	}
	return nil
}

// CountGroupAdmins returns how many admins a group has.
func (db *DB) CountGroupAdmins(ctx context.Context, roomID string) (int, error) {
	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return 0, fmt.Errorf("invalid room UUID: %w", err)
	}

	var count int
	err = db.Pool.QueryRow(ctx,
		`SELECT COUNT(*)::int FROM room_members WHERE room_id = $1 AND is_admin = true`,
		roomUUID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count group admins: %w", err)
	}
	return count, nil
}

// GetUserByUsername fetches a user ID by username.
func (db *DB) GetUserByUsername(ctx context.Context, username string) (*UserSearchResult, error) {
	row := db.Pool.QueryRow(ctx,
		`SELECT id, username, email, avatar_url FROM users WHERE username = $1`, username)

	var u UserSearchResult
	var id uuid.UUID
	if err := row.Scan(&id, &u.Username, &u.Email, &u.AvatarURL); err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	u.ID = id.String()
	return &u, nil
}

// FindDirectRoom returns an existing DM room between two users, if any.
func (db *DB) FindDirectRoom(ctx context.Context, user1ID, user2ID string) (*RoomWithMeta, error) {
	u1, err := uuid.Parse(user1ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user UUID: %w", err)
	}
	u2, err := uuid.Parse(user2ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user UUID: %w", err)
	}

	row := db.Pool.QueryRow(ctx,
		`SELECT r.id, r.name, r.type, r.created_at
		 FROM rooms r
		 JOIN room_members rm1 ON rm1.room_id = r.id AND rm1.user_id = $1
		 JOIN room_members rm2 ON rm2.room_id = r.id AND rm2.user_id = $2
		 WHERE r.type = 'direct'
		 LIMIT 1`,
		u1, u2)

	var r RoomWithMeta
	var id uuid.UUID
	if err := row.Scan(&id, &r.Name, &r.Type, &r.CreatedAt); err != nil {
		return nil, err
	}
	r.ID = id.String()
	return &r, nil
}

// CreateDirectRoom creates a new DM room between two users.
func (db *DB) CreateDirectRoom(ctx context.Context, user1ID, user2ID, otherUsername, otherAvatarURL string) (*RoomWithMeta, error) {
	u1, err := uuid.Parse(user1ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user UUID: %w", err)
	}
	u2, err := uuid.Parse(user2ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user UUID: %w", err)
	}

	// Deterministic internal name for direct rooms
	name := directRoomName(user1ID, user2ID)
	roomID := uuid.New()

	tx, err := db.Pool.Begin(ctx)
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

	return &RoomWithMeta{
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

// GetUserContacts returns all unique user IDs that share a room (direct or group) with the given user.
func (db *DB) GetUserContacts(ctx context.Context, userID string) ([]string, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user UUID: %w", err)
	}

	rows, err := db.Pool.Query(ctx, `
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

// MentionMember holds minimal member info for @mention resolution.
type MentionMember struct {
	UserID   string
	Username string
}

// GetRoomMembersForMentions returns room type, name, and members for @mention handling.
func (db *DB) GetRoomMembersForMentions(ctx context.Context, roomID string) (roomType string, roomName string, members []MentionMember, err error) {
	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return "", "", nil, fmt.Errorf("invalid room UUID: %w", err)
	}

	room, err := db.Queries.GetRoomByID(ctx, roomUUID)
	if err != nil {
		return "", "", nil, fmt.Errorf("room not found: %w", err)
	}

	roomType = room.Type
	if roomType == "private" {
		roomType = "group"
	}
	roomName = room.Name

	profiles, err := db.GetRoomMembersWithAvatar(ctx, roomID)
	if err != nil {
		return "", "", nil, err
	}

	members = make([]MentionMember, 0, len(profiles))
	for _, p := range profiles {
		members = append(members, MentionMember{
			UserID:   p.ID,
			Username: p.Username,
		})
	}
	return roomType, roomName, members, nil
}
