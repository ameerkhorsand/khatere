package application

import (
	"context"
	"io"
	"time"

	"github.com/bLorax/khatere-backend/internal/archive/domain"
	"github.com/google/uuid"
)

type UploadMediaUseCase struct {
	archives domain.ArchiveRepository
	media    domain.ArchiveMediaRepository
	hangouts domain.HangoutGateway
	storage  domain.MediaStorage
}

func NewUploadMediaUseCase(
	archives domain.ArchiveRepository,
	media domain.ArchiveMediaRepository,
	hangouts domain.HangoutGateway,
	storage domain.MediaStorage,
) *UploadMediaUseCase {
	return &UploadMediaUseCase{archives: archives, media: media, hangouts: hangouts, storage: storage}
}

type UploadMediaInput struct {
	ArchiveID  uuid.UUID
	UploaderID uuid.UUID
	MediaType  domain.MediaType
	Filename   string
	Content    io.Reader
	SizeBytes  int64

	// DurationSeconds is required when MediaType is video or
	// audio, and ignored for photo and gif.
	DurationSeconds *int
}

// Execute checks the archive is open, the uploader is a participant
// of the underlying hangout, and the file respects the media rules,
// then stores the file and records it.
func (uc *UploadMediaUseCase) Execute(ctx context.Context, in UploadMediaInput) (*domain.ArchiveMedia, error) {
	archive, err := uc.archives.FindByID(ctx, in.ArchiveID)
	if err != nil {
		return nil, err
	}
	if archive.IsPurged() {
		return nil, domain.ErrArchiveIsPurged
	}

	isParticipant, err := uc.hangouts.IsParticipant(ctx, archive.HangoutID, in.UploaderID)
	if err != nil {
		return nil, err
	}
	if !isParticipant {
		return nil, domain.ErrNotArchiveParticipant
	}

	if !in.MediaType.Valid() {
		return nil, domain.ErrInvalidMediaType
	}

	var duration *int
	if in.MediaType.HasDurationLimit() {
		if in.DurationSeconds == nil {
			return nil, domain.ErrInvalidMediaDuration
		}
		if err := domain.ValidateDuration(in.MediaType, *in.DurationSeconds); err != nil {
			return nil, err
		}
		duration = in.DurationSeconds
	}

	storageKey, err := uc.storage.Upload(ctx, archive.ID, in.MediaType, in.Filename, in.Content, in.SizeBytes)
	if err != nil {
		return nil, err
	}

	mediaRecord := &domain.ArchiveMedia{
		ID:              uuid.New(),
		ArchiveID:       archive.ID,
		UploaderID:      in.UploaderID,
		MediaType:       in.MediaType,
		StorageKey:      storageKey,
		DurationSeconds: duration,
		CreatedAt:       time.Now(),
	}

	if err := uc.media.Create(ctx, mediaRecord); err != nil {
		return nil, err
	}

	return mediaRecord, nil
}
