package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// MaxMediaDurationSeconds is the hard cap for video and audio clips.
const MaxMediaDurationSeconds = 60

// -----------------------------------------------------------------
// Archive
// -----------------------------------------------------------------

// ArchiveStatus tells if an archive is still available or has been
// purged after every participant asked to delete it.
type ArchiveStatus string

const (
	ArchiveStatusActive ArchiveStatus = "active"
	ArchiveStatusPurged ArchiveStatus = "purged"
)

func (s ArchiveStatus) Valid() bool {
	switch s {
	case ArchiveStatusActive, ArchiveStatusPurged:
		return true
	default:
		return false
	}
}

// Archive is a snapshot of one resolved Hangout: its chat log plus any
// media the participants added. It is created once the Hangout status
// becomes Completed or Cancelled.
type Archive struct {
	ID           uuid.UUID
	HangoutID    uuid.UUID
	ChatSnapshot string // rendered chat log at the time of creation
	Status       ArchiveStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (a *Archive) IsPurged() bool {
	return a.Status == ArchiveStatusPurged
}

// -----------------------------------------------------------------
// ArchiveMedia
// -----------------------------------------------------------------

type MediaType string

const (
	MediaTypePhoto MediaType = "photo"
	MediaTypeGIF   MediaType = "gif"
	MediaTypeVideo MediaType = "video"
	MediaTypeAudio MediaType = "audio"
)

func (t MediaType) Valid() bool {
	switch t {
	case MediaTypePhoto, MediaTypeGIF, MediaTypeVideo, MediaTypeAudio:
		return true
	default:
		return false
	}
}

// HasDurationLimit reports whether this media type must be checked
// against MaxMediaDurationSeconds.
func (t MediaType) HasDurationLimit() bool {
	return t == MediaTypeVideo || t == MediaTypeAudio
}

// ArchiveMedia is one uploaded file attached to an Archive.
type ArchiveMedia struct {
	ID              uuid.UUID
	ArchiveID       uuid.UUID
	UploaderID      uuid.UUID
	MediaType       MediaType
	StorageKey      string // object key in MinIO
	DurationSeconds *int   // set for video and audio, nil for photo and gif
	CreatedAt       time.Time
	DeletedAt       *time.Time
}

func (m *ArchiveMedia) IsDeleted() bool {
	return m.DeletedAt != nil
}

// ValidateDuration checks a video/audio clip against the max length.
// Call this before saving an ArchiveMedia record.
func ValidateDuration(mediaType MediaType, durationSeconds int) error {
	if !mediaType.HasDurationLimit() {
		return nil
	}
	if durationSeconds <= 0 {
		return ErrInvalidMediaDuration
	}
	if durationSeconds > MaxMediaDurationSeconds {
		return ErrMediaTooLong
	}
	return nil
}

// -----------------------------------------------------------------
// ArchiveDeletionMark
// -----------------------------------------------------------------

// ArchiveDeletionMark records that one participant asked to delete an
// archive. The archive is only purged from the server once every
// participant of the underlying hangout has left a mark.
type ArchiveDeletionMark struct {
	ArchiveID uuid.UUID
	UserID    uuid.UUID
	CreatedAt time.Time
}

// -----------------------------------------------------------------
// Domain errors
// -----------------------------------------------------------------

var (
	ErrArchiveNotFound       = errors.New("archive not found")
	ErrArchiveAlreadyExists  = errors.New("archive already exists for this hangout")
	ErrArchiveIsPurged       = errors.New("archive has been deleted by all participants")
	ErrHangoutNotResolved    = errors.New("archive can only be created for a completed or cancelled hangout")
	ErrInvalidMediaType      = errors.New("invalid media type")
	ErrInvalidMediaDuration  = errors.New("media duration must be a positive number of seconds")
	ErrMediaTooLong          = errors.New("video and audio clips must be 60 seconds or less")
	ErrNotArchiveParticipant = errors.New("only hangout participants can access this archive")
	ErrAlreadyMarkedDeleted  = errors.New("user has already marked this archive for deletion")
)
