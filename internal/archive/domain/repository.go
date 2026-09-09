package domain

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
)

// -----------------------------------------------------------------
// ArchiveRepository / ArchiveMediaRepository / DeletionMarkRepository
// -----------------------------------------------------------------

type ArchiveRepository interface {
	Create(ctx context.Context, archive *Archive) error

	FindByID(ctx context.Context, id uuid.UUID) (*Archive, error)

	// FindByHangoutID returns ErrArchiveNotFound if no archive has
	// been created yet for this hangout.
	FindByHangoutID(ctx context.Context, hangoutID uuid.UUID) (*Archive, error)

	Update(ctx context.Context, archive *Archive) error
}

type ArchiveMediaRepository interface {
	Create(ctx context.Context, media *ArchiveMedia) error

	// ListByArchive returns non-deleted media, oldest first.
	ListByArchive(ctx context.Context, archiveID uuid.UUID) ([]ArchiveMedia, error)

	SoftDelete(ctx context.Context, mediaID uuid.UUID) error
}

type DeletionMarkRepository interface {
	// Add inserts one mark. Returns ErrAlreadyMarkedDeleted if the
	// user already marked this archive.
	Add(ctx context.Context, mark *ArchiveDeletionMark) error

	// Count returns how many participants have marked this archive
	// for deletion so far.
	Count(ctx context.Context, archiveID uuid.UUID) (int, error)
}

// Transactor lets a use case group several archive-repository writes
// into one atomic transaction, the same pattern as
// internal/hangout/domain.Transactor. Used by DeleteArchiveUseCase to
// run the deletion mark and the unanimous-delete check together.
type Transactor interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// -----------------------------------------------------------------
// HangoutGateway
//
// The archive module never imports internal/hangout/domain
// directly — every other module in this codebase keeps its
// application layer scoped to its own domain package, and archive
// follows the same rule. Instead, archive declares the small slice
// of hangout data it needs as its own port, and an adapter (built
// in Step 9, living under internal/archive/adapters/hangout/) wraps
// the hangout module's real repositories to satisfy it.
// -----------------------------------------------------------------

// ResolvedHangoutStatus mirrors the two final values of the hangout
// module's own HangoutStatus. It is redeclared here, at the port
// boundary, rather than imported.
type ResolvedHangoutStatus string

const (
	ResolvedHangoutCompleted ResolvedHangoutStatus = "completed"
	ResolvedHangoutCancelled ResolvedHangoutStatus = "cancelled"
)

// ResolvedHangout is the subset of hangout data the archive module
// needs to create an archive.
type ResolvedHangout struct {
	ID             uuid.UUID
	Status         ResolvedHangoutStatus
	ScheduledEndAt *time.Time
	ParticipantIDs []uuid.UUID
}

func (h *ResolvedHangout) IsResolved() bool {
	return h.Status == ResolvedHangoutCompleted || h.Status == ResolvedHangoutCancelled
}

// ChatLine is one rendered line of a hangout's chat, decoupled from
// the hangout module's own Message type.
type ChatLine struct {
	SenderID  uuid.UUID
	Content   string
	CreatedAt time.Time
}

type HangoutGateway interface {
	// GetResolved returns ErrHangoutNotResolved if the hangout's
	// status is still Planned or Ongoing.
	GetResolved(ctx context.Context, hangoutID uuid.UUID) (*ResolvedHangout, error)

	// ListChatLog returns every message for the hangout, oldest
	// first.
	ListChatLog(ctx context.Context, hangoutID uuid.UUID) ([]ChatLine, error)

	// IsParticipant reports whether userID has a participant row
	// on hangoutID (organizer or invitee, any invite status). Used
	// to gate read/write access to an archive.
	IsParticipant(ctx context.Context, hangoutID, userID uuid.UUID) (bool, error)

	// ListResolvedForUser returns every Completed or Cancelled
	// hangout where userID is a participant, most recent first.
	// Used to compute the post-hangout upload-prompt window.
	ListResolvedForUser(ctx context.Context, userID uuid.UUID) ([]ResolvedHangout, error)
}

// -----------------------------------------------------------------
// MediaStorage
//
// Port over the object store (MinIO). Kept out of the application
// layer so use cases depend on this small interface, not on the
// MinIO SDK directly. The adapter (Step 9) wraps
// internal/platform/minio.
// -----------------------------------------------------------------

type MediaStorage interface {
	// Upload stores the file and returns the storage key to save
	// on the ArchiveMedia record.
	Upload(ctx context.Context, archiveID uuid.UUID, mediaType MediaType, filename string, content io.Reader, sizeBytes int64) (storageKey string, err error)

	// Delete removes a previously uploaded file. Used by the purge
	// use case (Step 7).
	Delete(ctx context.Context, storageKey string) error
}
