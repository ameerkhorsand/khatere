package http

import (
	"errors"
	"net/http"

	"github.com/bLorax/khatere-backend/internal/qrcode/application"
	"github.com/bLorax/khatere-backend/internal/qrcode/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handlers struct {
	generateQRCode *application.GenerateQRCodeUseCase
	getQRCode      *application.GetQRCodeUseCase
}

func NewHandlers(generateQRCode *application.GenerateQRCodeUseCase, getQRCode *application.GetQRCodeUseCase) *Handlers {
	return &Handlers{generateQRCode: generateQRCode, getQRCode: getQRCode}
}

// RegisterRoutes is mounted under the activity group in server.go
// (e.g. activityGroup.Group("/:id")), so id is read from the parent
// route's :id param, same as the rating domain.
func (h *Handlers) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/qrcode", h.GenerateQRCode)
	r.GET("/qrcode", h.GetQRCode)
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

func (h *Handlers) GenerateQRCode(c *gin.Context) {
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

	qr, err := h.generateQRCode.Execute(c.Request.Context(), application.GenerateQRCodeInput{
		ActivityID: activityID,
		HostID:     hostID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotHostActivity):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusOK, qr)
}

func (h *Handlers) GetQRCode(c *gin.Context) {
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

	qr, err := h.getQRCode.Execute(c.Request.Context(), application.GetQRCodeInput{
		ActivityID: activityID,
		HostID:     hostID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotHostActivity):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case errors.Is(err, domain.ErrQRCodeNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusOK, qr)
}
