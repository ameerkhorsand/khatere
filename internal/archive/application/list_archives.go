package application

import (
	"context"
	"errors"

	"github.com/bLorax/khatere-backend/internal/archive/domain"
	"github.com/google/uuid"
)

type ListArchivesUseCase struct {
	archives domain.ArchiveRepository
	hangouts domain.HangoutGateway
}

func NewListArchivesUseCase(archives domain.ArchiveRepository, hangouts domain.HangoutGateway) *ListArchivesUseCase {
	return &ListArchivesUseCase{archives: archives, hangouts: hangouts}
}

type ListArchivesInput struct {
	UserID uuid.UUID
}

// Execute returns one archive per resolved hangout the caller took
// part in. A hangout with no archive yet (should not normally
// happen, since CreateArchiveUseCase runs as soon as a hangout
// resolves) is silently skipped rather than treated as an error.
//
// Authorization: this use case has no separate participant check
// because ListResolvedForUser(in.UserID) already scopes the query to
// hangouts where UserID is a participant — there is no path for a
// caller to see another user's archive through this method. Do not
// change this to take a HangoutID parameter without adding an
// IsParticipant check, the way GetArchiveUseCase does.
func (uc *ListArchivesUseCase) Execute(ctx context.Context, in ListArchivesInput) ([]domain.Archive, error) {
	resolvedHangouts, err := uc.hangouts.ListResolvedForUser(ctx, in.UserID)
	if err != nil {
		return nil, err
	}

	archives := make([]domain.Archive, 0, len(resolvedHangouts))
	for _, h := range resolvedHangouts {
		archive, err := uc.archives.FindByHangoutID(ctx, h.ID)
		if err != nil {
			if errors.Is(err, domain.ErrArchiveNotFound) {
				continue
			}
			return nil, err
		}
		archives = append(archives, *archive)
	}

	return archives, nil
}
