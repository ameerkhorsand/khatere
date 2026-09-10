package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// UserBadge is one badge earned by a user. It is made right after
// a verified scan, when the activity has a badge set up.
//
// NameSnapshot and IconKeySnapshot hold a copy of the badge's
// fields at award time. BadgeID and ActivityID can later go to
// nil if the host edits, removes the badge, or removes the
// activity — the snapshot fields never change, so the user keeps
// what they earned.
type UserBadge struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	BadgeID         *uuid.UUID // nil if the badge was later removed
	ActivityID      *uuid.UUID // nil if the activity was later removed
	NameSnapshot    string
	IconKeySnapshot string
	Visible         bool // user's own show/hide choice for their profile
	AwardedAt       time.Time
}

var ErrUserBadgeNotFound = errors.New("user badge not found")
