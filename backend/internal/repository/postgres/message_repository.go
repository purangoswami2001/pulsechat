package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pulsechat/backend/internal/db/postgres/sqlc"
	"github.com/pulsechat/backend/internal/domain"
)

type MessageRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewMessageRepository(pool *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *MessageRepository) Create(ctx context.Context, id, roomID, senderID, content string) (*domain.Message, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid message uuid: %w", err)
	}
	rID, err := uuid.Parse(roomID)
	if err != nil {
		return nil, fmt.Errorf("invalid room uuid: %w", err)
	}
	sID, err := uuid.Parse(senderID)
	if err != nil {
		return nil, fmt.Errorf("invalid sender uuid: %w", err)
	}

	params := sqlc.CreateMessageParams{
		ID:       uid,
		RoomID:   rID,
		SenderID: sID,
		Content:  content,
	}

	row, err := r.queries.CreateMessage(ctx, params)
	if err != nil {
		return nil, err
	}

	return &domain.Message{
		ID:        row.ID.String(),
		RoomID:    row.RoomID.String(),
		SenderID:  row.SenderID.String(),
		Content:   row.Content,
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

func (r *MessageRepository) CreateWithAttachment(ctx context.Context, id, roomID, senderID, content, attachmentURL, attachmentType string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid message uuid: %w", err)
	}
	rID, err := uuid.Parse(roomID)
	if err != nil {
		return fmt.Errorf("invalid room uuid: %w", err)
	}
	sID, err := uuid.Parse(senderID)
	if err != nil {
		return fmt.Errorf("invalid sender uuid: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO messages (id, room_id, sender_id, content, attachment_url, attachment_type, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP)`,
		uid, rID, sID, content, attachmentURL, attachmentType)
	if err != nil {
		return fmt.Errorf("failed to create message with attachment: %w", err)
	}
	return nil
}

func (r *MessageRepository) ListWithAttachments(ctx context.Context, roomID string, limit, offset int) ([]domain.Message, error) {
	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return nil, fmt.Errorf("invalid room UUID: %w", err)
	}

	rows, err := r.pool.Query(ctx,
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

	var messages []domain.Message
	for rows.Next() {
		var m domain.Message
		var id, rID, sID uuid.UUID
		if err := rows.Scan(&id, &rID, &sID, &m.SenderName, &m.SenderAvatarURL, &m.Content,
			&m.AttachmentURL, &m.AttachmentType, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}
		m.ID = id.String()
		m.RoomID = rID.String()
		m.SenderID = sID.String()
		messages = append(messages, m)
	}

	if messages == nil {
		messages = []domain.Message{}
	}
	return messages, nil
}
