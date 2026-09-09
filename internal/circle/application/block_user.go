package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/circle/domain"
	"github.com/google/uuid"
)

type BlockUserUseCase struct {
	blocks domain.BlockRepository
}

func NewBlockUserUseCase(
	blocks domain.BlockRepository,
) *BlockUserUseCase {
	return &BlockUserUseCase{
		blocks: blocks,
	}
}

type BlockUserInput struct {
	BlockerID uuid.UUID
	BlockedID uuid.UUID
}

func (uc *BlockUserUseCase) Execute(
	ctx context.Context,
	in BlockUserInput,
) (*domain.Block, error) {
	if in.BlockerID == in.BlockedID {
		return nil, domain.ErrCannotBlockSelf
	}

	return uc.blocks.Block(
		ctx,
		in.BlockerID,
		in.BlockedID,
	)
}
