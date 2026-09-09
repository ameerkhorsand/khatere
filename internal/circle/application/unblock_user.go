package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/circle/domain"
	"github.com/google/uuid"
)

type UnblockUserUseCase struct {
	blocks domain.BlockRepository
}

func NewUnblockUserUseCase(
	blocks domain.BlockRepository,
) *UnblockUserUseCase {
	return &UnblockUserUseCase{
		blocks: blocks,
	}
}

type UnblockUserInput struct {
	BlockerID uuid.UUID
	BlockedID uuid.UUID
}

func (uc *UnblockUserUseCase) Execute(
	ctx context.Context,
	in UnblockUserInput,
) error {
	return uc.blocks.Unblock(
		ctx,
		in.BlockerID,
		in.BlockedID,
	)
}
