package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Badge is the one badge a host sets up by hand for an activity.
// The in-app badge design tool is deferred, so a host fills in
// Name and IconKey directly (through a moderator/admin-assisted
// flow for now).
type Badge struct {
	ID         uuid.UUID
	ActivityID uuid.UUID
	CreatedBy  uuid.UUID // FK to accounts.id (the host)
	Name       string
	IconKey    string // object storage key, not a URL — same
	// pattern as User.ProfilePictureKey. Generate a
	// presigned URL on read.
	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
}

var (
	ErrBadgeNotFound   = errors.New("badge not found")
	ErrNotHostActivity = errors.New("only the host who created this activity may manage its badge")
	// ErrBadgeAlreadyExists guards the one-badge-per-activity rule
	// at the application layer, ahead of the UNIQUE(activity_id)
	// constraint (migration 0018). A badge's fields matter to the
	// host, so a second create attempt is rejected instead of
	// silently returning the first badge (unlike qrcode's Generate,
	// which is safe to repeat because the code itself never changes).
	ErrBadgeAlreadyExists = errors.New("this activity already has a badge")
)
