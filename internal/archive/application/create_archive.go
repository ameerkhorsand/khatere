package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bLorax/khatere-backend/internal/archive/domain"
	"github.com/google/uuid"
)

type CreateArchiveUseCase struct {
	archives domain.ArchiveRepository
	hangouts domain.HangoutGateway
}

func NewCreateArchiveUseCase(archives domain.ArchiveRepository, hangouts domain.HangoutGateway) *CreateArchiveUseCase {
	return &CreateArchiveUseCase{archives: archives, hangouts: hangouts}
}

type CreateArchiveInput struct {
	HangoutID uuid.UUID
}

// Execute is called once a hangout's status becomes Completed or
// Cancelled. See internal/hangout/application/update_hangout_status.go
// and cancel_hangout.go for the two places that should trigger this
// (Step 9 wires that call). It builds one Archive row holding a
// rendered snapshot of the hangout's chat log.
func (uc *CreateArchiveUseCase) Execute(ctx context.Context, in CreateArchiveInput) (*domain.Archive, error) {
	existing, err := uc.archives.FindByHangoutID(ctx, in.HangoutID)
	if err != nil && !errors.Is(err, domain.ErrArchiveNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrArchiveAlreadyExists
	}

	hangout, err := uc.hangouts.GetResolved(ctx, in.HangoutID)
	if err != nil {
		return nil, err
	}
	if !hangout.IsResolved() {
		return nil, domain.ErrHangoutNotResolved
	}

	lines, err := uc.hangouts.ListChatLog(ctx, in.HangoutID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	archive := &domain.Archive{
		ID:           uuid.New(),
		HangoutID:    hangout.ID,
		ChatSnapshot: renderChatSnapshot(lines),
		Status:       domain.ArchiveStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := uc.archives.Create(ctx, archive); err != nil {
		return nil, err
	}

	return archive, nil
}

// renderChatSnapshot turns the chat log into one plain-text block,
// one line per message, oldest first.
func renderChatSnapshot(lines []domain.ChatLine) string {
	var b strings.Builder
	for _, l := range lines {
		fmt.Fprintf(&b, "[%s] %s: %s\n", l.CreatedAt.Format(time.RFC3339), l.SenderID, l.Content)
	}
	return b.String()
}
