package chat

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pulsechat/backend/internal/db"
)

// SQLCMessageStore adapts the SQLC database Queries layer to the MessageStore interface.
type SQLCMessageStore struct {
	db *db.DB
}

// NewSQLCMessageStore creates a new message store adapter instance.
func NewSQLCMessageStore(database *db.DB) *SQLCMessageStore {
	return &SQLCMessageStore{db: database}
}

// Save implements the MessageStore Save method, including attachment fields.
func (s *SQLCMessageStore) Save(ctx context.Context, msg Message) error {
	msgID, err := uuid.Parse(msg.ID)
	if err != nil {
		return fmt.Errorf("invalid message UUID format: %w", err)
	}

	roomID, err := uuid.Parse(msg.RoomID)
	if err != nil {
		return fmt.Errorf("invalid room UUID format: %w", err)
	}

	senderID, err := uuid.Parse(msg.SenderID)
	if err != nil {
		return fmt.Errorf("invalid sender UUID format: %w", err)
	}

	// Use the manual query that includes attachment columns
	return s.db.CreateMessageWithAttachment(ctx, msgID, roomID, senderID,
		msg.Content, msg.AttachmentURL, msg.AttachmentType)
}

// ListByRoom implements the MessageStore ListByRoom method with pagination.
func (s *SQLCMessageStore) ListByRoom(ctx context.Context, roomID string, limit, offset int) ([]Message, error) {
	rows, err := s.db.ListMessagesWithAttachments(ctx, roomID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages from database: %w", err)
	}

	messages := make([]Message, len(rows))
	for i, row := range rows {
		messages[i] = Message{
			ID:             row.ID,
			RoomID:         row.RoomID,
			SenderID:       row.SenderID,
			SenderName:     row.SenderName,
			Content:        row.Content,
			AttachmentURL:  row.AttachmentURL,
			AttachmentType: row.AttachmentType,
			CreatedAt:      row.CreatedAt,
		}
	}

	return messages, nil
}
