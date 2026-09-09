package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/hangout/domain"
	"github.com/google/uuid"
)

type ListHangoutsUseCase struct {
	hangouts domain.HangoutRepository
}

func NewListHangoutsUseCase(hangouts domain.HangoutRepository) *ListHangoutsUseCase {
	return &ListHangoutsUseCase{hangouts: hangouts}
}

type ListHangoutsInput struct {
	UserID uuid.UUID
	// Status nil = all tabs combined. Set it to filter to one of the
	// four Hangouts List tabs (Upcoming=planned, Ongoing, Completed,
	// Didn't Happen=cancelled).
	Status *domain.HangoutStatus
}

func (uc *ListHangoutsUseCase) Execute(ctx context.Context, in ListHangoutsInput) ([]domain.Hangout, error) {
	return uc.hangouts.ListByUser(ctx, in.UserID, domain.ListFilter{Status: in.Status})
}
