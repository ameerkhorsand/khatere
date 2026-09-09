package application

import (
	"context"
	"time"

	"github.com/bLorax/khatere-backend/internal/hangout/domain"
	"github.com/google/uuid"
)

const defaultMessagePageSize = 50
const maxMessagePageSize = 100

type ListMessagesUseCase struct {
	participants domain.ParticipantRepository
	messages     domain.MessageRepository
}

func NewListMessagesUseCase(participants domain.ParticipantRepository, messages domain.MessageRepository) *ListMessagesUseCase {
	return &ListMessagesUseCase{participants: participants, messages: messages}
}

type ListMessagesInput struct {
	HangoutID   uuid.UUID
	RequesterID uuid.UUID
	Before      *time.Time
	Limit       int
}

// Execute requires the requester to already have a row in
// hangout_participants — pending invitees can read the chat to help
// them decide, but a stranger cannot.
func (uc *ListMessagesUseCase) Execute(ctx context.Context, in ListMessagesInput) ([]domain.Message, error) {
	if _, err := uc.participants.FindParticipant(ctx, in.HangoutID, in.RequesterID); err != nil {
		return nil, err
	}

	limit := in.Limit
	if limit <= 0 {
		limit = defaultMessagePageSize
	}
	if limit > maxMessagePageSize {
		limit = maxMessagePageSize
	}

	return uc.messages.ListMessages(ctx, in.HangoutID, domain.MessageFilter{
		Before: in.Before,
		Limit:  limit,
	})
}
