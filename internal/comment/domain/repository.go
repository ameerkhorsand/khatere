package domain

import (
	"context"

	"github.com/google/uuid"
)

type CommentRepository interface {
	Create(ctx context.Context, comment *Comment) error

	FindByID(ctx context.Context, id uuid.UUID) (*Comment, error)

	// ListApprovedByActivity returns approved, non-deleted comments
	// for an activity. Ordering (by rank) is applied by the use
	// case, not here — see ListCommentsUseCase.
	ListApprovedByActivity(ctx context.Context, activityID uuid.UUID) ([]Comment, error)

	// ListPendingQueue returns pending, non-deleted comments across
	// all activities, oldest first — same fair-queue ordering as
	// ActivityRepository.ListPendingQueue.
	ListPendingQueue(ctx context.Context) ([]Comment, error)

	// Update performs an optimistic-lock update: checks
	// comment.Version against the stored row, returns
	// ErrVersionConflict on mismatch.
	Update(ctx context.Context, comment *Comment) error
}

// VoteTally is the up/down count for one comment, used by
// ListCommentsUseCase to rank approved comments.
type VoteTally struct {
	Up   int
	Down int
}

type CommentVoteRepository interface {
	// Upsert inserts a vote, or overwrites the existing vote for
	// (CommentID, UserID) if the user already voted — a user can
	// change their mind, not stack votes. See the PRIMARY KEY on
	// comment_votes in migration 0016.
	Upsert(ctx context.Context, vote *CommentVote) error

	FindVote(ctx context.Context, commentID, userID uuid.UUID) (*CommentVote, error)

	// TallyForComments returns vote counts for each comment in
	// commentIDs in one query, keyed by comment ID. A comment with
	// no votes yet is simply absent from the result — callers
	// treat a missing key as VoteTally{0, 0}.
	TallyForComments(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID]VoteTally, error)
}

// -----------------------------------------------------------------
// CommentSummarizer
//
// Port over the AI summary provider (DeepSeek). Kept out of the
// application layer so the use case depends on this one method, not
// on any particular vendor's SDK or HTTP shape — same pattern as
// domain.MediaStorage in the archive module. The adapter lives at
// internal/comment/adapters/deepseek.
// -----------------------------------------------------------------

type CommentSummarizer interface {
	// Summarize takes the approved comment bodies for one activity,
	// oldest first, and returns a short natural-language summary.
	Summarize(ctx context.Context, commentBodies []string) (string, error)
}
