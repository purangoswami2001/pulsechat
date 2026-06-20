package mysql

import (
	"context"
	"errors"

	"github.com/pulsechat/backend/internal/domain"
)

type MessageRepository struct{}

func NewMessageRepository() *MessageRepository {
	return &MessageRepository{}
}

func (r *MessageRepository) Create(ctx context.Context, id, roomID, senderID, content string) (*domain.Message, error) {
	return nil, errors.New("mysql repository is not implemented")
}

func (r *MessageRepository) CreateWithAttachment(ctx context.Context, id, roomID, senderID, content, attachmentURL, attachmentType string) error {
	return errors.New("mysql repository is not implemented")
}

func (r *MessageRepository) ListWithAttachments(ctx context.Context, roomID string, limit, offset int) ([]domain.Message, error) {
	return nil, errors.New("mysql repository is not implemented")
}
