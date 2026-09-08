package application

import (
	"context"
	"time"

	"github.com/bLorax/khatere-backend/internal/host/domain"
	"github.com/google/uuid"
)

type CreateHostProfileUseCase struct {
	hosts domain.HostRepository
}

func NewCreateHostProfileUseCase(hosts domain.HostRepository) *CreateHostProfileUseCase {
	return &CreateHostProfileUseCase{hosts: hosts}
}

type CreateHostProfileInput struct {
	AccountID    uuid.UUID
	BusinessName string
	LocationInfo *string
}

func (uc *CreateHostProfileUseCase) Execute(ctx context.Context, in CreateHostProfileInput) (*domain.Host, error) {
	existing, err := uc.hosts.FindByAccountID(ctx, in.AccountID)
	if err != nil && err != domain.ErrHostNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrHostAlreadyExists
	}

	host := &domain.Host{
		AccountID:    in.AccountID,
		BusinessName: in.BusinessName,
		LocationInfo: in.LocationInfo,
		Metadata:     map[string]any{},
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := uc.hosts.Create(ctx, host); err != nil {
		return nil, err
	}

	return host, nil
}
