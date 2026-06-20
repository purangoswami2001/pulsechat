package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/pulsechat/backend/internal/domain"
	"github.com/pulsechat/backend/internal/repository"
)

type MessageService struct {
	msgRepo repository.MessageRepository
}

func NewMessageService(msgRepo repository.MessageRepository) *MessageService {
	return &MessageService{msgRepo: msgRepo}
}

func (s *MessageService) SaveMessage(ctx context.Context, msg domain.Message) (*domain.Message, error) {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}

	if msg.AttachmentURL != "" {
		err := s.msgRepo.CreateWithAttachment(ctx, msg.ID, msg.RoomID, msg.SenderID, msg.Content, msg.AttachmentURL, msg.AttachmentType)
		if err != nil {
			return nil, err
		}
		return &msg, nil
	}

	return s.msgRepo.Create(ctx, msg.ID, msg.RoomID, msg.SenderID, msg.Content)
}

func (s *MessageService) ListMessages(ctx context.Context, roomID string, limit, offset int) ([]domain.Message, error) {
	return s.msgRepo.ListWithAttachments(ctx, roomID, limit, offset)
}
