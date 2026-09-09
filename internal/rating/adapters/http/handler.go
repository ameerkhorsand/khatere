package http

import (
	"errors"
	"net/http"

	"github.com/bLorax/khatere-backend/internal/rating/application"
	"github.com/bLorax/khatere-backend/internal/rating/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handlers struct {
	createRating     *application.CreateRatingUseCase
	getRatingSummary *application.GetRatingSummaryUseCase
}

func NewHandlers(
	createRating *application.CreateRatingUseCase,
	getRatingSummary *application.GetRatingSummaryUseCase,
) *Handlers {
	return &Handlers{createRating: createRating, getRatingSummary: getRatingSummary}
}

// RegisterRoutes is mounted under the activity group in server.go
// (e.g. r.Group("/activity/:id")), so id is read from the parent
// route's :id param, same as GetActivity in the activity domain.
func (h *Handlers) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/rating", h.CreateRating)
	r.GET("/rating", h.GetRatingSummary)
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

type createRatingRequest struct {
	// Score is a pointer so an explicit 0 (a valid rating) is
	// distinguishable from a missing field — binding:"required"
	// on a plain float64 would reject 0 as if it were absent.
	Score *float64 `json:"score" binding:"required"`
}

func (h *Handlers) CreateRating(c *gin.Context) {
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

	var req createRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rating, err := h.createRating.Execute(c.Request.Context(), application.CreateRatingInput{
		ActivityID: activityID,
		UserID:     userID,
		Score:      *req.Score,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidScore):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, domain.ErrNotAttendee):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusCreated, rating)
}

func (h *Handlers) GetRatingSummary(c *gin.Context) {
	activityID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid activity id"})
		return
	}

	summary, err := h.getRatingSummary.Execute(c.Request.Context(), activityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, summary)
}
