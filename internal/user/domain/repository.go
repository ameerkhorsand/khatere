package domain

import (
	"context"
	"io"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByAccountID(ctx context.Context, accountID uuid.UUID) (*User, error)
	FindByHandle(ctx context.Context, handle string) (*User, error)

	// Update performs an optimistic-lock update: it checks user.Version
	// against the stored row and returns ErrVersionConflict on mismatch.
	Update(ctx context.Context, user *User) error
}

type InterestRepository interface {
	// ListActive returns the full catalog of active interests, for onboarding UI.
	ListActive(ctx context.Context) ([]Interest, error)

	// ReplaceAll deletes the user's existing interest selections and inserts
	// the given interest IDs, as a single transaction.
	ReplaceAll(ctx context.Context, userID uuid.UUID, interestIDs []uuid.UUID) error

	// FindByUserID returns the full Interest records the user has selected.
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]Interest, error)
}

// -----------------------------------------------------------------
// ProfileStorage
//
// Port over the object store (MinIO), for profile pictures only.
// Kept out of the application layer so use cases depend on this
// small interface, not on the MinIO SDK directly. Mirrors
// internal/archive/domain.MediaStorage.
// -----------------------------------------------------------------

type ProfileStorage interface {
	// Upload stores the file and returns the storage key to save
	// on the user's ProfilePictureKey field.
	Upload(ctx context.Context, accountID uuid.UUID, filename string, contentType string, content io.Reader, sizeBytes int64) (storageKey string, err error)

	// Delete removes a previously uploaded file. Used when a user
	// replaces an existing profile picture. Callers treat failure
	// as best-effort — see UploadProfilePictureUseCase.
	Delete(ctx context.Context, storageKey string) error

	// PublicURL turns a storage key into a URL the frontend can
	// open directly. The URL is time-limited — do not store it,
	// generate it fresh on each read.
	PublicURL(ctx context.Context, storageKey string) (string, error)
}

// MaxProfilePictureSizeBytes is the hard cap enforced by
// UploadProfilePictureUseCase before any upload is attempted.
const MaxProfilePictureSizeBytes = 10 * 1024 * 1024 // 10 MB

// allowedProfilePictureTypes are the only content types accepted
// for a profile picture upload.
var allowedProfilePictureTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

// ValidContentType reports whether contentType is one of the
// allowed profile picture formats (jpg, png, webp).
func ValidContentType(contentType string) bool {
	return allowedProfilePictureTypes[contentType]
}
