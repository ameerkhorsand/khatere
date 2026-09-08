package domain

import (
	"context"

	"github.com/google/uuid"
)

type HostRepository interface {
	Create(ctx context.Context, host *Host) error
	FindByAccountID(ctx context.Context, accountID uuid.UUID) (*Host, error)
	Update(ctx context.Context, host *Host) error
}
