package application

import (
	"context"
	"time"

	"github.com/bLorax/khatere-backend/internal/archive/domain"
	"github.com/google/uuid"
)

type CheckAndPurgeArchiveUseCase struct {
	archives domain.ArchiveRepository
	media    domain.ArchiveMediaRepository
	hangouts domain.HangoutGateway
	marks    domain.DeletionMarkRepository
	storage  domain.MediaStorage
}

func NewCheckAndPurgeArchiveUseCase(
	archives domain.ArchiveRepository,
	media domain.ArchiveMediaRepository,
	hangouts domain.HangoutGateway,
	marks domain.DeletionMarkRepository,
	storage domain.MediaStorage,
) *CheckAndPurgeArchiveUseCase {
	return &CheckAndPurgeArchiveUseCase{archives: archives, media: media, hangouts: hangouts, marks: marks, storage: storage}
}

type CheckAndPurgeArchiveInput struct {
	ArchiveID uuid.UUID
}

// Execute is meant to run right after MarkArchiveDeletedUseCase adds
// a new mark (Step 10 wires the two calls together in the HTTP
// handler). It compares the number of deletion marks against the
// hangout's participant count. Nothing happens until every
// participant has marked the archive; at that point every media file
// is removed from storage and the archive is flipped to Purged.
//
// The archive row and the deletion marks are kept after a purge, as
// an audit trail — only the media files and the chat_snapshot text
// are removed. Returns true if a purge happened.
//
// Authorization: this use case takes no UserID and does not check
// participation itself — it trusts that its only caller,
// DeleteArchiveUseCase, has already verified the requester through
// MarkArchiveDeletedUseCase before this runs. Do not call this use
// case directly from an HTTP handler without adding that check back.
func (uc *CheckAndPurgeArchiveUseCase) Execute(ctx context.Context, in CheckAndPurgeArchiveInput) (bool, error) {
	archive, err := uc.archives.FindByID(ctx, in.ArchiveID)
	if err != nil {
		return false, err
	}
	if archive.IsPurged() {
		return false, nil
	}

	hangout, err := uc.hangouts.GetResolved(ctx, archive.HangoutID)
	if err != nil {
		return false, err
	}

	markCount, err := uc.marks.Count(ctx, archive.ID)
	if err != nil {
		return false, err
	}

	if markCount < len(hangout.ParticipantIDs) {
		return false, nil // not unanimous yet
	}

	files, err := uc.media.ListByArchive(ctx, archive.ID)
	if err != nil {
		return false, err
	}

	for _, m := range files {
		if err := uc.storage.Delete(ctx, m.StorageKey); err != nil {
			return false, err
		}
		if err := uc.media.SoftDelete(ctx, m.ID); err != nil {
			return false, err
		}
	}

	archive.Status = domain.ArchiveStatusPurged
	archive.ChatSnapshot = ""
	archive.UpdatedAt = time.Now()

	if err := uc.archives.Update(ctx, archive); err != nil {
		return false, err
	}

	return true, nil
}
