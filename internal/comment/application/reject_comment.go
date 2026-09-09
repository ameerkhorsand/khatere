package application

import (
	"context"
	"time"

	"github.com/bLorax/khatere-backend/internal/comment/domain"
	"github.com/google/uuid"
)

type RejectCommentUseCase struct {
	comments domain.CommentRepository
}

func NewRejectCommentUseCase(comments domain.CommentRepository) *RejectCommentUseCase {
	return &RejectCommentUseCase{comments: comments}
}

type RejectCommentInput struct {
	CommentID   uuid.UUID
	ModeratorID uuid.UUID
}

func (uc *RejectCommentUseCase) Execute(ctx context.Context, in RejectCommentInput) (*domain.Comment, error) {
	comment, err := uc.comments.FindByID(ctx, in.CommentID)
	if err != nil {
		return nil, err
	}

	if comment.Status != domain.CommentStatusPending {
		return nil, domain.ErrCommentNotPending
	}

	now := time.Now()
	comment.Status = domain.CommentStatusRejected
	comment.ReviewedBy = &in.ModeratorID
	comment.ReviewedAt = &now

	if err := uc.comments.Update(ctx, comment); err != nil {
		return nil, err
	}

	return comment, nil
}
