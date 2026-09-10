package application

import (
	"context"
	"sort"

	"github.com/bLorax/khatere-backend/internal/comment/domain"
	"github.com/google/uuid"
)

// VoteWeight is the vote portion of the ranking formula (70%, per
// the project spec). AttendanceBoostWeight is reserved for Phase 7,
// once attendance verification exists — see the doc comment on
// computeScore below for exactly where it plugs in.
const (
	VoteWeight            = 0.70
	AttendanceBoostWeight = 0.30 // unused until Phase 7
)

// AttendanceVerificationChecker is intentionally small so this use
// case doesn't depend on the concrete attendance repository — same
// pattern as AttendanceChecker in the rating domain.
// AttendanceVerificationRepository.IsVerified (internal/attendance/adapters/postgres)
// satisfies this interface.
type AttendanceVerificationChecker interface {
	IsVerified(ctx context.Context, activityID, userID uuid.UUID) (bool, error)
}

// RankedComment pairs a comment with the vote data its rank was
// computed from, so the HTTP layer can show vote counts without a
// second query.
type RankedComment struct {
	Comment   domain.Comment
	Score     float64
	Upvotes   int
	Downvotes int
}

type ListCommentsUseCase struct {
	comments   domain.CommentRepository
	votes      domain.CommentVoteRepository
	attendance AttendanceVerificationChecker
}

func NewListCommentsUseCase(comments domain.CommentRepository, votes domain.CommentVoteRepository, attendance AttendanceVerificationChecker) *ListCommentsUseCase {
	return &ListCommentsUseCase{comments: comments, votes: votes, attendance: attendance}
}

// Execute returns approved comments for an activity, ranked highest
// score first. Only the vote-weighted side of the formula is
// applied here (attendance verification is Phase 7's addition).
func (uc *ListCommentsUseCase) Execute(ctx context.Context, activityID uuid.UUID) ([]RankedComment, error) {
	comments, err := uc.comments.ListApprovedByActivity(ctx, activityID)
	if err != nil {
		return nil, err
	}
	if len(comments) == 0 {
		return []RankedComment{}, nil
	}

	ids := make([]uuid.UUID, len(comments))
	for i, c := range comments {
		ids[i] = c.ID
	}

	tallies, err := uc.votes.TallyForComments(ctx, ids)
	if err != nil {
		return nil, err
	}
	ranked := make([]RankedComment, len(comments))
	for i, c := range comments {
		tally := tallies[c.ID] // zero value {0, 0} for comments absent from the map

		verified, err := uc.attendance.IsVerified(ctx, activityID, c.UserID)
		if err != nil {
			return nil, err
		}

		ranked[i] = RankedComment{
			Comment:   c,
			Score:     computeScore(tally, verified),
			Upvotes:   tally.Up,
			Downvotes: tally.Down,
		}
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Score != ranked[j].Score {
			return ranked[i].Score > ranked[j].Score
		}
		// Tie-break: more recent comment first.
		return ranked[i].Comment.CreatedAt.After(ranked[j].Comment.CreatedAt)
	})

	return ranked, nil
}

// computeScore is Phase 6's version of the ranking formula: net
// votes, weighted at 70%. Phase 7 adds a second term here —
// something like `+ AttendanceBoostWeight` when the comment's
// author is a verified attendee — without touching anything else
// in this file.
//
// computeScore is the full Phase 7 ranking formula: net votes,
// weighted at 70%, plus a flat boost when the comment's author has
// a verified attendance record for this activity.
func computeScore(tally domain.VoteTally, verified bool) float64 {
	net := float64(tally.Up - tally.Down)
	score := net * VoteWeight
	if verified {
		score += AttendanceBoostWeight
	}
	return score
}
