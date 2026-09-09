package application

import (
	"context"
	"time"

	"github.com/bLorax/khatere-backend/internal/archive/domain"
	"github.com/google/uuid"
)

type MarkArchiveDeletedUseCase struct {
	archives domain.ArchiveRepository
	hangouts domain.HangoutGateway
	marks    domain.DeletionMarkRepository
}

func NewMarkArchiveDeletedUseCase(
	archives domain.ArchiveRepository,
	hangouts domain.HangoutGateway,
	marks domain.DeletionMarkRepository,
) *MarkArchiveDeletedUseCase {
	return &MarkArchiveDeletedUseCase{archives: archives, hangouts: hangouts, marks: marks}
}

type MarkArchiveDeletedInput struct {
	ArchiveID uuid.UUID
	UserID    uuid.UUID
}

// Execute records that UserID wants this archive deleted for
// themselves. This is a personal flag only: it hides the archive for
// this user but does not remove any data. The unanimous-delete check
// that decides whether to purge the archive from the server is a
// separate step (see purge_archive.go), run right after this mark is
// added.
func (uc *MarkArchiveDeletedUseCase) Execute(ctx context.Context, in MarkArchiveDeletedInput) error {
	archive, err := uc.archives.FindByID(ctx, in.ArchiveID)
	if err != nil {
		return err
	}
	if archive.IsPurged() {
		return domain.ErrArchiveIsPurged
	}

	isParticipant, err := uc.hangouts.IsParticipant(ctx, archive.HangoutID, in.UserID)
	if err != nil {
		return err
	}
	if !isParticipant {
		return domain.ErrNotArchiveParticipant
	}

	mark := &domain.ArchiveDeletionMark{
		ArchiveID: archive.ID,
		UserID:    in.UserID,
		CreatedAt: time.Now(),
	}

	return uc.marks.Add(ctx, mark)
}
