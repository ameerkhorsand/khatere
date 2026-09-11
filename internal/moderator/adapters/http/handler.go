package http

import (
	"errors"
	"net/http"

	"github.com/bLorax/khatere-backend/internal/moderator/application"
	"github.com/bLorax/khatere-backend/internal/moderator/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handlers struct {
	createModeratorProfile *application.CreateModeratorProfileUseCase
	getModeratorProfile    *application.GetModeratorProfileUseCase
}

func NewHandlers(createModeratorProfile *application.CreateModeratorProfileUseCase, getModeratorProfile *application.GetModeratorProfileUseCase) *Handlers {
	return &Handlers{createModeratorProfile: createModeratorProfile, getModeratorProfile: getModeratorProfile}
}

func (h *Handlers) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/profile", h.CreateProfile)
	r.GET("/profile", h.GetProfile)
}

func (h *Handlers) CreateProfile(c *gin.Context) {
	raw, exists := c.Get("account_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	accountID, err := uuid.Parse(raw.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	moderator, err := h.createModeratorProfile.Execute(c.Request.Context(), application.CreateModeratorProfileInput{
		AccountID: accountID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrModeratorAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "moderator profile already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusCreated, moderator)
}

func (h *Handlers) GetProfile(c *gin.Context) {
	raw, exists := c.Get("account_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	accountID, err := uuid.Parse(raw.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	moderator, err := h.getModeratorProfile.Execute(c.Request.Context(), accountID)
	if err != nil {
		if errors.Is(err, domain.ErrModeratorNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "moderator profile not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, moderator)
}
