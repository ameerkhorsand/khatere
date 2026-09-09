package application

import (
	"context"
	"time"

	"github.com/bLorax/khatere-backend/internal/comment/domain"
	"github.com/google/uuid"
)

type VoteCommentUseCase struct {
	comments domain.CommentRepository
	votes    domain.CommentVoteRepository
}

func NewVoteCommentUseCase(comments domain.CommentRepository, votes domain.CommentVoteRepository) *VoteCommentUseCase {
	return &VoteCommentUseCase{comments: comments, votes: votes}
}

type VoteCommentInput struct {
	CommentID uuid.UUID
	UserID    uuid.UUID
	Value     domain.VoteValue
}

// Execute records or replaces a user's vote on a comment. Only
// approved comments can be voted on — a pending or rejected comment
// isn't shown to anyone yet, so there's nothing to vote on.
func (uc *VoteCommentUseCase) Execute(ctx context.Context, in VoteCommentInput) error {
	if !in.Value.Valid() {
		return domain.ErrInvalidVoteValue
	}

	comment, err := uc.comments.FindByID(ctx, in.CommentID)
	if err != nil {
		return err
	}
	if !comment.IsVisible() {
		return domain.ErrCommentNotFound
	}

	vote := &domain.CommentVote{
		CommentID: in.CommentID,
		UserID:    in.UserID,
		Value:     in.Value,
		CreatedAt: time.Now(),
	}

	return uc.votes.Upsert(ctx, vote)
}
