package http

import (
	"errors"
	"net/http"

	"github.com/bLorax/khatere-backend/internal/activity/application"
	"github.com/bLorax/khatere-backend/internal/activity/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handlers struct {
	createActivity        *application.CreateActivityUseCase
	getActivity           *application.GetActivityUseCase
	listActivities        *application.ListActivitiesUseCase
	listMyActivities      *application.ListMyActivitiesUseCase
	listModerationQueue   *application.ListModerationQueueUseCase
	approveActivity       *application.ApproveActivityUseCase
	rejectActivity        *application.RejectActivityUseCase
	setActivityInterests  *application.SetActivityInterestsUseCase
	listActivityInterests *application.ListActivityInterestsUseCase
}

func NewHandlers(
	createActivity *application.CreateActivityUseCase,
	getActivity *application.GetActivityUseCase,
	listActivities *application.ListActivitiesUseCase,
	listMyActivities *application.ListMyActivitiesUseCase,
	listModerationQueue *application.ListModerationQueueUseCase,
	approveActivity *application.ApproveActivityUseCase,
	rejectActivity *application.RejectActivityUseCase,
	setActivityInterests *application.SetActivityInterestsUseCase,
	listActivityInterests *application.ListActivityInterestsUseCase,
) *Handlers {
	return &Handlers{
		createActivity, getActivity, listActivities, listMyActivities,
		listModerationQueue, approveActivity, rejectActivity,
		setActivityInterests, listActivityInterests,
	}
}

func (h *Handlers) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("", h.CreateActivity)
	r.GET("", h.ListActivities)
	r.GET("/mine", h.ListMyActivities)
	r.GET("/:id", h.GetActivity)

	r.GET("/moderation/queue", h.ListModerationQueue)
	r.POST("/:id/approve", h.ApproveActivity)
	r.POST("/:id/reject", h.RejectActivity)

	r.PUT("/:id/interests", h.SetActivityInterests)
	r.GET("/:id/interests", h.ListActivityInterests)
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

// accountTypeFromContext reads the caller's account type, set by the JWT
// middleware alongside account_id.
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

type createActivityRequest struct {
	Title       string  `json:"title" binding:"required"`
	Description *string `json:"description"`
}

func (h *Handlers) CreateActivity(c *gin.Context) {
	accountID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	accountType, err := accountTypeFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req createActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	activity, err := h.createActivity.Execute(c.Request.Context(), application.CreateActivityInput{
		Title:       req.Title,
		Description: req.Description,
		SourceType:  domain.SourceType(accountType),
		CreatedBy:   accountID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidSourceType) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account type for creating an activity"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusCreated, activity)
}

func (h *Handlers) GetActivity(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid activity id"})
		return
	}

	activity, err := h.getActivity.Execute(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrActivityNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "activity not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, activity)
}

func (h *Handlers) ListActivities(c *gin.Context) {
	activities, err := h.listActivities.Execute(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, activities)
}

func (h *Handlers) ListMyActivities(c *gin.Context) {
	accountID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	activities, err := h.listMyActivities.Execute(c.Request.Context(), accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, activities)
}

func (h *Handlers) ListModerationQueue(c *gin.Context) {
	if !requireModerator(c) {
		return
	}

	activities, err := h.listModerationQueue.Execute(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, activities)
}

func (h *Handlers) ApproveActivity(c *gin.Context) {
	if !requireModerator(c) {
		return
	}

	moderatorID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	activityID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid activity id"})
		return
	}

	activity, err := h.approveActivity.Execute(c.Request.Context(), application.ApproveActivityInput{
		ActivityID:  activityID,
		ModeratorID: moderatorID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrActivityNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "activity not found"})
		case errors.Is(err, domain.ErrActivityNotPending):
			c.JSON(http.StatusConflict, gin.H{"error": "activity is not pending review"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusOK, activity)
}

type setActivityInterestsRequest struct {
	InterestIDs []string `json:"interest_ids" binding:"required"`
}

func (h *Handlers) SetActivityInterests(c *gin.Context) {
	callerID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	accountType, err := accountTypeFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	activityID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid activity id"})
		return
	}

	var req setActivityInterestsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	interestIDs := make([]uuid.UUID, 0, len(req.InterestIDs))
	for _, raw := range req.InterestIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid interest id: " + raw})
			return
		}
		interestIDs = append(interestIDs, id)
	}

	err = h.setActivityInterests.Execute(c.Request.Context(), application.SetActivityInterestsInput{
		ActivityID:        activityID,
		InterestIDs:       interestIDs,
		CallerID:          callerID,
		CallerIsModerator: accountType == "moderator",
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrActivityNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "activity not found"})
		case errors.Is(err, domain.ErrNotAuthorizedToTag):
			c.JSON(http.StatusForbidden, gin.H{"error": "only the activity's creator or a moderator can tag its interests"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handlers) ListActivityInterests(c *gin.Context) {
	activityID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid activity id"})
		return
	}

	interests, err := h.listActivityInterests.Execute(c.Request.Context(), activityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, interests)
}

type rejectActivityRequest struct {
	Reason *string `json:"reason"`
}

func (h *Handlers) RejectActivity(c *gin.Context) {
	if !requireModerator(c) {
		return
	}

	moderatorID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	activityID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid activity id"})
		return
	}

	var req rejectActivityRequest
	_ = c.ShouldBindJSON(&req) // reason is optional, ignore bind errors on empty body

	activity, err := h.rejectActivity.Execute(c.Request.Context(), application.RejectActivityInput{
		ActivityID:  activityID,
		ModeratorID: moderatorID,
		Reason:      req.Reason,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrActivityNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "activity not found"})
		case errors.Is(err, domain.ErrActivityNotPending):
			c.JSON(http.StatusConflict, gin.H{"error": "activity is not pending review"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusOK, activity)
}
