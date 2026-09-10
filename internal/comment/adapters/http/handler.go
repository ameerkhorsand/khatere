package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/bLorax/khatere-backend/internal/comment/application"
	"github.com/bLorax/khatere-backend/internal/comment/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handlers struct {
	createComment       *application.CreateCommentUseCase
	listComments        *application.ListCommentsUseCase
	voteComment         *application.VoteCommentUseCase
	listModerationQueue *application.ListCommentModerationQueueUseCase
	approveComment      *application.ApproveCommentUseCase
	rejectComment       *application.RejectCommentUseCase
	getSummary          *application.GetCommentSummaryUseCase
}

func NewHandlers(
	createComment *application.CreateCommentUseCase,
	listComments *application.ListCommentsUseCase,
	voteComment *application.VoteCommentUseCase,
	listModerationQueue *application.ListCommentModerationQueueUseCase,
	approveComment *application.ApproveCommentUseCase,
	rejectComment *application.RejectCommentUseCase,
	getSummary *application.GetCommentSummaryUseCase,
) *Handlers {
	return &Handlers{
		createComment, listComments, voteComment,
		listModerationQueue, approveComment, rejectComment,
		getSummary,
	}
}

// RegisterRoutes is mounted under the activity group in server.go
// (r.Group("/:id"), same as the rating handler), so :id is the
// activity's id. Produces POST/GET /activities/:id/comment and
// GET /activities/:id/comment/summary.
func (h *Handlers) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/comment", h.CreateComment)
	r.GET("/comment", h.ListComments)
	r.GET("/comment/summary", h.GetSummary)
}

// RegisterVoteRoute is mounted on its own top-level group in
// server.go (router.Group("/comment")), since a vote targets a
// comment directly, not an activity. Produces POST /comment/:id/vote.
func (h *Handlers) RegisterVoteRoute(r *gin.RouterGroup) {
	r.POST("/:id/vote", h.VoteComment)
}

// RegisterModerationRoutes is mounted on the existing moderator
// group in server.go. Produces GET /moderator/comment/queue,
// POST /moderator/comment/:id/approve, POST /moderator/comment/:id/reject
// — same shape as the activity domain's moderation routes.
func (h *Handlers) RegisterModerationRoutes(r *gin.RouterGroup) {
	r.GET("/comment/queue", h.ListModerationQueue)
	r.POST("/comment/:id/approve", h.ApproveComment)
	r.POST("/comment/:id/reject", h.RejectComment)
}

func accountIDFromContext(c *gin.Context) (uuid.UUID, error) {
	raw, exists := c.Get("account_id")
	if !exists {
		return uuid.UUID{}, errors.New("no account_id in context")
	}
	id, err := uuid.Parse(raw.(string))
	if err != nil {
		return uuid.UUID{}, errors.New("invalid account_id in context")
	}
	return id, nil
}

func accountTypeFromContext(c *gin.Context) (string, error) {
	raw, exists := c.Get("account_type")
	if !exists {
		return "", errors.New("no account_type in context")
	}
	return raw.(string), nil
}

func requireModerator(c *gin.Context) bool {
	accountType, err := accountTypeFromContext(c)
	if err != nil || accountType != "moderator" {
		c.JSON(http.StatusForbidden, gin.H{"error": "moderator account required"})
		return false
	}
	return true
}

type createCommentRequest struct {
	Body string `json:"body" binding:"required"`
}

func (h *Handlers) CreateComment(c *gin.Context) {
	userID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	activityID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid activity id"})
		return
	}

	var req createCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	comment, err := h.createComment.Execute(c.Request.Context(), application.CreateCommentInput{
		ActivityID: activityID,
		UserID:     userID,
		Body:       req.Body,
	})
	if err != nil {
		if errors.Is(err, domain.ErrEmptyBody) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusCreated, comment)
}

func (h *Handlers) ListComments(c *gin.Context) {
	activityID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid activity id"})
		return
	}

	comments, err := h.listComments.Execute(c.Request.Context(), activityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, comments)
}

type voteCommentRequest struct {
	// Plain int, not a pointer: unlike Rating's Score, 0 is never a
	// legitimate vote value (only 1 or -1 are), so binding:"required"
	// rejecting a missing/zero field is exactly the behavior we want.
	Value int `json:"value" binding:"required"`
}

func (h *Handlers) VoteComment(c *gin.Context) {
	userID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	commentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment id"})
		return
	}

	var req voteCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.voteComment.Execute(c.Request.Context(), application.VoteCommentInput{
		CommentID: commentID,
		UserID:    userID,
		Value:     domain.VoteValue(req.Value),
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidVoteValue):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, domain.ErrCommentNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "recorded"})
}

func (h *Handlers) ListModerationQueue(c *gin.Context) {
	if !requireModerator(c) {
		return
	}

	comments, err := h.listModerationQueue.Execute(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, comments)
}

func (h *Handlers) ApproveComment(c *gin.Context) {
	if !requireModerator(c) {
		return
	}

	moderatorID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	commentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment id"})
		return
	}

	comment, err := h.approveComment.Execute(c.Request.Context(), application.ApproveCommentInput{
		CommentID:   commentID,
		ModeratorID: moderatorID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCommentNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
		case errors.Is(err, domain.ErrCommentNotPending):
			c.JSON(http.StatusConflict, gin.H{"error": "comment is not pending review"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusOK, comment)
}

func (h *Handlers) RejectComment(c *gin.Context) {
	if !requireModerator(c) {
		return
	}

	moderatorID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	commentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment id"})
		return
	}

	comment, err := h.rejectComment.Execute(c.Request.Context(), application.RejectCommentInput{
		CommentID:   commentID,
		ModeratorID: moderatorID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCommentNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
		case errors.Is(err, domain.ErrCommentNotPending):
			c.JSON(http.StatusConflict, gin.H{"error": "comment is not pending review"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusOK, comment)
}

// commentSummaryResponse is a deliberate, explicit shape rather
// than returning domain.CommentSummary directly — "available" is
// what a frontend actually needs to branch on (show the summary vs.
// show nothing), and it doesn't exist as a field on the domain type.
type commentSummaryResponse struct {
	Available    bool       `json:"available"`
	Summary      string     `json:"summary"`
	CommentCount int        `json:"comment_count"`
	GeneratedAt  *time.Time `json:"generated_at,omitempty"`
}

func (h *Handlers) GetSummary(c *gin.Context) {
	activityID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid activity id"})
		return
	}

	summary, err := h.getSummary.Execute(c.Request.Context(), activityID)
	if err != nil {
		if errors.Is(err, domain.ErrSummaryNotFound) {
			// Not an error condition — every activity starts here,
			// before its first comment is ever approved. 200 with
			// available:false lets the frontend show "no summary
			// yet" instead of treating a normal state as a failure.
			c.JSON(http.StatusOK, commentSummaryResponse{Available: false})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, commentSummaryResponse{
		Available:    true,
		Summary:      summary.Summary,
		CommentCount: summary.CommentCount,
		GeneratedAt:  summary.GeneratedAt,
	})
}
