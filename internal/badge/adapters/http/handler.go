package http

import (
	"errors"
	"net/http"

	"github.com/bLorax/khatere-backend/internal/badge/application"
	"github.com/bLorax/khatere-backend/internal/badge/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handlers struct {
	createBadge *application.CreateBadgeUseCase
	getBadge    *application.GetBadgeUseCase
}

func NewHandlers(createBadge *application.CreateBadgeUseCase, getBadge *application.GetBadgeUseCase) *Handlers {
	return &Handlers{createBadge: createBadge, getBadge: getBadge}
}

// RegisterRoutes is mounted under the activity group in server.go
// (e.g. activityGroup.Group("/:id")), so id is read from the parent
// route's :id param, same as the qrcode domain.
func (h *Handlers) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/badge", h.CreateBadge)
	r.GET("/badge", h.GetBadge)
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

type createBadgeRequest struct {
	Name    string `json:"name" binding:"required"`
	IconKey string `json:"icon_key"`
}

func (h *Handlers) CreateBadge(c *gin.Context) {
	hostID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	activityID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid activity id"})
		return
	}

	var req createBadgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	badge, err := h.createBadge.Execute(c.Request.Context(), application.CreateBadgeInput{
		ActivityID: activityID,
		HostID:     hostID,
		Name:       req.Name,
		IconKey:    req.IconKey,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotHostActivity):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case errors.Is(err, domain.ErrBadgeAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusOK, badge)
}

func (h *Handlers) GetBadge(c *gin.Context) {
	hostID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	activityID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid activity id"})
		return
	}

	badge, err := h.getBadge.Execute(c.Request.Context(), application.GetBadgeInput{
		ActivityID: activityID,
		HostID:     hostID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotHostActivity):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case errors.Is(err, domain.ErrBadgeNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusOK, badge)
}
