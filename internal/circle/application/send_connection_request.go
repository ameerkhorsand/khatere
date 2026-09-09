package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/circle/domain"
	"github.com/google/uuid"
)

type SendConnectionRequestUseCase struct {
	connections domain.ConnectionRepository
	blocks      domain.BlockRepository
}

func NewSendConnectionRequestUseCase(
	connections domain.ConnectionRepository,
	blocks domain.BlockRepository,
) *SendConnectionRequestUseCase {
	return &SendConnectionRequestUseCase{
		connections: connections,
		blocks:      blocks,
	}
}

type SendConnectionRequestInput struct {
	RequesterID uuid.UUID
	AddresseeID uuid.UUID
}

func (uc *SendConnectionRequestUseCase) Execute(
	ctx context.Context,
	in SendConnectionRequestInput,
) (*domain.Connection, error) {
	if in.RequesterID == in.AddresseeID {
		return nil, domain.ErrCannotConnectSelf
	}

	blockedByRequester, err := uc.blocks.IsBlocked(
		ctx,
		in.RequesterID,
		in.AddresseeID,
	)
	if err != nil {
		return nil, err
	}

	blockedByAddressee, err := uc.blocks.IsBlocked(
		ctx,
		in.AddresseeID,
		in.RequesterID,
	)
	if err != nil {
		return nil, err
	}

	if blockedByRequester || blockedByAddressee {
		return nil, domain.ErrBlocked
	}

	return uc.connections.CreateRequest(
		ctx,
		in.RequesterID,
		in.AddresseeID,
	)
}
