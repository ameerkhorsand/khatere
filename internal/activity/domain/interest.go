package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Interest mirrors user/domain.Interest field-for-field. It is
// duplicated here rather than imported, matching this codebase's
// existing rule that one domain never imports another domain's
// package directly (see badge/qrcode's ActivityOwnershipChecker
// pattern) — both domains read the same interests lookup table,
// they just don't share a Go type for it.
type Interest struct {
	ID        uuid.UUID
	Slug      string
	Label     string
	Category  *string
	Active    bool
	SortOrder int
	CreatedAt time.Time
}

// ActivityInterestRepository manages which interests an activity is
// tagged with (activity_interests, migration 0020). This lives in
// the activity domain, not the recommendation domain, because
// tagging is something an activity's creator or a moderator does as
// part of managing the activity — the recommendation engine only
// ever reads this table, never writes it.
type ActivityInterestRepository interface {
	// ReplaceAll overwrites activityID's full tag set in one
	// transaction, same reasoning as user/domain.InterestRepository.ReplaceAll:
	// a caller sends the whole desired set, not one tag at a time.
	ReplaceAll(ctx context.Context, activityID uuid.UUID, interestIDs []uuid.UUID) error

	FindByActivityID(ctx context.Context, activityID uuid.UUID) ([]Interest, error)
}
