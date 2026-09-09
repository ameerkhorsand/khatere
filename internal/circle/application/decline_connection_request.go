package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/circle/domain"
	"github.com/google/uuid"
)

type DeclineConnectionRequestUseCase struct {
	connections domain.ConnectionRepository
}

func NewDeclineConnectionRequestUseCase(
	connections domain.ConnectionRepository,
) *DeclineConnectionRequestUseCase {
	return &DeclineConnectionRequestUseCase{
		connections: connections,
	}
}

type DeclineConnectionRequestInput struct {
	RequestID   uuid.UUID
	AddresseeID uuid.UUID
}

func (uc *DeclineConnectionRequestUseCase) Execute(
	ctx context.Context,
	in DeclineConnectionRequestInput,
) error {
	return uc.connections.Decline(
		ctx,
		in.RequestID,
		in.AddresseeID,
	)
}
