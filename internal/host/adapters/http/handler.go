package http

import (
	"errors"
	"net/http"

	"github.com/bLorax/khatere-backend/internal/host/application"
	"github.com/bLorax/khatere-backend/internal/host/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handlers struct {
	createHostProfile *application.CreateHostProfileUseCase
}

func NewHandlers(createHostProfile *application.CreateHostProfileUseCase) *Handlers {
	return &Handlers{createHostProfile: createHostProfile}
}

func (h *Handlers) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/profile", h.CreateProfile)
}

type createHostProfileRequest struct {
	BusinessName string  `json:"business_name" binding:"required"`
	LocationInfo *string `json:"location_info"`
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

	var req createHostProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	host, err := h.createHostProfile.Execute(c.Request.Context(), application.CreateHostProfileInput{
		AccountID:    accountID,
		BusinessName: req.BusinessName,
		LocationInfo: req.LocationInfo,
	})
	if err != nil {
		if errors.Is(err, domain.ErrHostAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "host profile already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusCreated, host)
}
