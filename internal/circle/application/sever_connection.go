package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/circle/domain"
	"github.com/google/uuid"
)

type SeverConnectionUseCase struct {
	connections domain.ConnectionRepository
}

func NewSeverConnectionUseCase(
	connections domain.ConnectionRepository,
) *SeverConnectionUseCase {
	return &SeverConnectionUseCase{
		connections: connections,
	}
}

type SeverConnectionInput struct {
	ConnectionID uuid.UUID
	UserID       uuid.UUID
}

func (uc *SeverConnectionUseCase) Execute(
	ctx context.Context,
	in SeverConnectionInput,
) error {
	return uc.connections.Sever(
		ctx,
		in.ConnectionID,
		in.UserID,
	)
}
