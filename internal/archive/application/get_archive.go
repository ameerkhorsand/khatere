package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/archive/domain"
	"github.com/google/uuid"
)

type GetArchiveUseCase struct {
	archives domain.ArchiveRepository
	media    domain.ArchiveMediaRepository
	hangouts domain.HangoutGateway
}

func NewGetArchiveUseCase(archives domain.ArchiveRepository, media domain.ArchiveMediaRepository, hangouts domain.HangoutGateway) *GetArchiveUseCase {
	return &GetArchiveUseCase{archives: archives, media: media, hangouts: hangouts}
}

type GetArchiveInput struct {
	ArchiveID   uuid.UUID
	RequesterID uuid.UUID
}

// ArchiveDetail bundles an archive with its media list — the
// Archive Detail screen needs both.
type ArchiveDetail struct {
	Archive domain.Archive
	Media   []domain.ArchiveMedia
}

// Execute only returns the archive to a participant of the
// underlying hangout.
func (uc *GetArchiveUseCase) Execute(ctx context.Context, in GetArchiveInput) (*ArchiveDetail, error) {
	archive, err := uc.archives.FindByID(ctx, in.ArchiveID)
	if err != nil {
		return nil, err
	}

	isParticipant, err := uc.hangouts.IsParticipant(ctx, archive.HangoutID, in.RequesterID)
	if err != nil {
		return nil, err
	}
	if !isParticipant {
		return nil, domain.ErrNotArchiveParticipant
	}

	media, err := uc.media.ListByArchive(ctx, archive.ID)
	if err != nil {
		return nil, err
	}

	return &ArchiveDetail{Archive: *archive, Media: media}, nil
}
