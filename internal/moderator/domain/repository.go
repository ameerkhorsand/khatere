package domain

import (
	"context"

	"github.com/google/uuid"
)

type ModeratorRepository interface {
	Create(ctx context.Context, moderator *Moderator) error
	FindByAccountID(ctx context.Context, accountID uuid.UUID) (*Moderator, error)
	Update(ctx context.Context, moderator *Moderator) error
}
