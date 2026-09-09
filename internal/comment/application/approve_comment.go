package application

import (
	"context"
	"time"

	"github.com/bLorax/khatere-backend/internal/comment/domain"
	"github.com/google/uuid"
)

type ApproveCommentUseCase struct {
	comments domain.CommentRepository
}

func NewApproveCommentUseCase(comments domain.CommentRepository) *ApproveCommentUseCase {
	return &ApproveCommentUseCase{comments: comments}
}

type ApproveCommentInput struct {
	CommentID   uuid.UUID
	ModeratorID uuid.UUID
}

func (uc *ApproveCommentUseCase) Execute(ctx context.Context, in ApproveCommentInput) (*domain.Comment, error) {
	comment, err := uc.comments.FindByID(ctx, in.CommentID)
	if err != nil {
		return nil, err
	}

	if comment.Status != domain.CommentStatusPending {
		return nil, domain.ErrCommentNotPending
	}

	now := time.Now()
	comment.Status = domain.CommentStatusApproved
	comment.ReviewedBy = &in.ModeratorID
	comment.ReviewedAt = &now

	if err := uc.comments.Update(ctx, comment); err != nil {
		return nil, err
	}

	return comment, nil
}
