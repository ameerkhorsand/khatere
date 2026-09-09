package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/circle/domain"
	"github.com/google/uuid"
)

type AcceptConnectionRequestUseCase struct {
	connections domain.ConnectionRepository
	blocks      domain.BlockRepository
}

func NewAcceptConnectionRequestUseCase(
	connections domain.ConnectionRepository,
	blocks domain.BlockRepository,
) *AcceptConnectionRequestUseCase {
	return &AcceptConnectionRequestUseCase{
		connections: connections,
		blocks:      blocks,
	}
}

type AcceptConnectionRequestInput struct {
	RequestID   uuid.UUID
	AddresseeID uuid.UUID
}

func (uc *AcceptConnectionRequestUseCase) Execute(
	ctx context.Context,
	in AcceptConnectionRequestInput,
) (*domain.Connection, error) {
	// We first fetch pending requests so we can enforce the block rule
	// before transitioning the request to accepted.
	requests, err := uc.connections.ListIncomingPending(ctx, in.AddresseeID)
	if err != nil {
		return nil, err
	}

	var request *domain.Connection
	for i := range requests {
		if requests[i].ID == in.RequestID {
			request = &requests[i]
			break
		}
	}

	if request == nil {
		return nil, domain.ErrRequestNotFound
	}

	blockedByAddressee, err := uc.blocks.IsBlocked(
		ctx,
		in.AddresseeID,
		request.RequesterID,
	)
	if err != nil {
		return nil, err
	}

	blockedByRequester, err := uc.blocks.IsBlocked(
		ctx,
		request.RequesterID,
		in.AddresseeID,
	)
	if err != nil {
		return nil, err
	}

	if blockedByAddressee || blockedByRequester {
		return nil, domain.ErrBlocked
	}

	return uc.connections.Accept(
		ctx,
		in.RequestID,
		in.AddresseeID,
	)
}
