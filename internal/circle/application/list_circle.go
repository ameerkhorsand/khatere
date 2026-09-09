package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/circle/domain"
	"github.com/google/uuid"
)

type ListCircleUseCase struct {
	connections domain.ConnectionRepository
}

func NewListCircleUseCase(
	connections domain.ConnectionRepository,
) *ListCircleUseCase {
	return &ListCircleUseCase{
		connections: connections,
	}
}

func (uc *ListCircleUseCase) Execute(
	ctx context.Context,
	userID uuid.UUID,
) ([]domain.Connection, error) {
	return uc.connections.ListCircle(ctx, userID)
}
