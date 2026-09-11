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
	getHostProfile    *application.GetHostProfileUseCase
}

func NewHandlers(createHostProfile *application.CreateHostProfileUseCase, getHostProfile *application.GetHostProfileUseCase) *Handlers {
	return &Handlers{createHostProfile: createHostProfile, getHostProfile: getHostProfile}
}

func (h *Handlers) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/profile", h.CreateProfile)
	r.GET("/profile", h.GetProfile)
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

	host, err := h.getHostProfile.Execute(c.Request.Context(), accountID)
	if err != nil {
		if errors.Is(err, domain.ErrHostNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "host profile not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, host)
}
