package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/bLorax/khatere-backend/internal/hangout/domain"
	"github.com/google/uuid"
)

const maxMessageLength = 2000

type SendMessageUseCase struct {
	participants domain.ParticipantRepository
	messages     domain.MessageRepository
}

func NewSendMessageUseCase(participants domain.ParticipantRepository, messages domain.MessageRepository) *SendMessageUseCase {
	return &SendMessageUseCase{participants: participants, messages: messages}
}

type SendMessageInput struct {
	HangoutID uuid.UUID
	SenderID  uuid.UUID
	Content   string
}

var (
	ErrEmptyMessage   = errors.New("message content cannot be empty")
	ErrMessageTooLong = errors.New("message content exceeds the maximum length")
)

func (uc *SendMessageUseCase) Execute(ctx context.Context, in SendMessageInput) (*domain.Message, error) {
	content := strings.TrimSpace(in.Content)
	if content == "" {
		return nil, ErrEmptyMessage
	}
	if len(content) > maxMessageLength {
		return nil, ErrMessageTooLong
	}

	participant, err := uc.participants.FindParticipant(ctx, in.HangoutID, in.SenderID)
	if err != nil {
		return nil, err
	}
	if participant.InviteStatus != domain.InviteStatusAccepted {
		return nil, domain.ErrNotAcceptedParticipant
	}

	message := &domain.Message{
		ID:        uuid.New(),
		HangoutID: in.HangoutID,
		SenderID:  in.SenderID,
		Content:   content,
		CreatedAt: time.Now(),
	}

	if err := uc.messages.CreateMessage(ctx, message); err != nil {
		return nil, err
	}

	return message, nil
}
