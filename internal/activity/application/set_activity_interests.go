package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/activity/domain"
	"github.com/google/uuid"
)

// SetActivityInterestsUseCase tags an activity with a set of interests,
// replacing whatever set it had before. Only the activity's creator or
// a moderator may do this.
//
// The permission check lives here rather than at the HTTP layer (unlike
// requireModerator's role-only check for ApproveActivity/RejectActivity):
// deciding "creator OR moderator" needs the activity's CreatedBy field,
// and this use case already loads the activity via FindByID for the
// existence check, so checking here avoids a second lookup.
type SetActivityInterestsUseCase struct {
	activities domain.ActivityRepository
	interests  domain.ActivityInterestRepository
}

func NewSetActivityInterestsUseCase(
	activities domain.ActivityRepository,
	interests domain.ActivityInterestRepository,
) *SetActivityInterestsUseCase {
	return &SetActivityInterestsUseCase{activities: activities, interests: interests}
}

type SetActivityInterestsInput struct {
	ActivityID        uuid.UUID
	InterestIDs       []uuid.UUID
	CallerID          uuid.UUID
	CallerIsModerator bool
}

func (uc *SetActivityInterestsUseCase) Execute(ctx context.Context, in SetActivityInterestsInput) error {
	activity, err := uc.activities.FindByID(ctx, in.ActivityID)
	if err != nil {
		return err
	}

	if activity.CreatedBy != in.CallerID && !in.CallerIsModerator {
		return domain.ErrNotAuthorizedToTag
	}

	// No need to validate that the interest IDs exist — the FK constraint
	// on activity_interests.interest_id catches a bad ID, mirroring
	// SetInterestsUseCase in the user domain.
	return uc.interests.ReplaceAll(ctx, in.ActivityID, in.InterestIDs)
}
