package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/circle/domain"
	"github.com/google/uuid"
)

type ListPendingRequestsUseCase struct {
	connections domain.ConnectionRepository
}

func NewListPendingRequestsUseCase(
	connections domain.ConnectionRepository,
) *ListPendingRequestsUseCase {
	return &ListPendingRequestsUseCase{
		connections: connections,
	}
}

type PendingRequests struct {
	Incoming []domain.Connection
	Outgoing []domain.Connection
}

func (uc *ListPendingRequestsUseCase) Execute(
	ctx context.Context,
	userID uuid.UUID,
) (*PendingRequests, error) {
	incoming, err := uc.connections.ListIncomingPending(ctx, userID)
	if err != nil {
		return nil, err
	}

	outgoing, err := uc.connections.ListOutgoingPending(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &PendingRequests{
		Incoming: incoming,
		Outgoing: outgoing,
	}, nil
}
