package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/comment/domain"
)

type ListCommentModerationQueueUseCase struct {
	comments domain.CommentRepository
}

func NewListCommentModerationQueueUseCase(comments domain.CommentRepository) *ListCommentModerationQueueUseCase {
	return &ListCommentModerationQueueUseCase{comments: comments}
}

func (uc *ListCommentModerationQueueUseCase) Execute(ctx context.Context) ([]domain.Comment, error) {
	return uc.comments.ListPendingQueue(ctx)
}
