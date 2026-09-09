package application

import (
	"context"
	"errors"
	"time"

	"github.com/bLorax/khatere-backend/internal/archive/domain"
	"github.com/google/uuid"
)

// UploadPromptWindow is how long after a hangout's scheduled end
// time participants may still be prompted to add media.
const UploadPromptWindow = 7 * 24 * time.Hour

type ListPendingUploadPromptsUseCase struct {
	hangouts domain.HangoutGateway
	archives domain.ArchiveRepository
}

func NewListPendingUploadPromptsUseCase(hangouts domain.HangoutGateway, archives domain.ArchiveRepository) *ListPendingUploadPromptsUseCase {
	return &ListPendingUploadPromptsUseCase{hangouts: hangouts, archives: archives}
}

type ListPendingUploadPromptsInput struct {
	UserID uuid.UUID
}

type PendingUploadPrompt struct {
	HangoutID      uuid.UUID
	ArchiveID      uuid.UUID
	WindowClosesAt time.Time
}

// Execute is the on-demand check the frontend calls (e.g. on app
// open) to decide whether to show the upload prompt. It lists the
// caller's resolved hangouts that are still inside the 1-week window
// after ScheduledEndAt, have an archive, and are not yet purged.
//
// This is Option B from Step 8 of the plan: an on-demand check
// instead of a background worker. No new infrastructure is needed;
// move to a polling worker later only if push notifications are
// required.
//
// Authorization: same model as ListArchivesUseCase — scoped by
// ListResolvedForUser(in.UserID), no separate participant check
// needed or present.
func (uc *ListPendingUploadPromptsUseCase) Execute(ctx context.Context, in ListPendingUploadPromptsInput) ([]PendingUploadPrompt, error) {
	resolved, err := uc.hangouts.ListResolvedForUser(ctx, in.UserID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	prompts := []PendingUploadPrompt{}

	for _, h := range resolved {
		if h.ScheduledEndAt == nil {
			continue // no fixed end time, no window to compute
		}

		windowClosesAt := h.ScheduledEndAt.Add(UploadPromptWindow)
		if now.After(windowClosesAt) {
			continue // window has closed
		}

		archive, err := uc.archives.FindByHangoutID(ctx, h.ID)
		if err != nil {
			if errors.Is(err, domain.ErrArchiveNotFound) {
				continue // archive not created yet, nothing to upload into
			}
			return nil, err
		}
		if archive.IsPurged() {
			continue // every participant already deleted it
		}

		prompts = append(prompts, PendingUploadPrompt{
			HangoutID:      h.ID,
			ArchiveID:      archive.ID,
			WindowClosesAt: windowClosesAt,
		})
	}

	return prompts, nil
}
