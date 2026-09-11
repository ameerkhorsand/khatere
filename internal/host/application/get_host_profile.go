package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/host/domain"
	"github.com/google/uuid"
)

type GetHostProfileUseCase struct {
	hosts domain.HostRepository
}

func NewGetHostProfileUseCase(hosts domain.HostRepository) *GetHostProfileUseCase {
	return &GetHostProfileUseCase{hosts: hosts}
}

// Execute returns the host profile for the given account ID.
// It returns domain.ErrHostNotFound when no profile exists yet.
func (uc *GetHostProfileUseCase) Execute(ctx context.Context, accountID uuid.UUID) (*domain.Host, error) {
	host, err := uc.hosts.FindByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return host, nil
}
