package http

import (
	"errors"
	"net/http"

	"github.com/bLorax/khatere-backend/internal/attendance/application"
	"github.com/bLorax/khatere-backend/internal/attendance/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handlers struct {
	verifyScan *application.VerifyScanUseCase
}

func NewHandlers(verifyScan *application.VerifyScanUseCase) *Handlers {
	return &Handlers{verifyScan: verifyScan}
}

// RegisterRoutes mounts a standalone /attendance group — unlike
// rating, qrcode, and badge, a scan does not carry an activity id
// in its URL. The scanned code is the only thing that names the
// activity, so it travels in the request body instead.
func (h *Handlers) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/scan", h.VerifyScan)
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

type verifyScanRequest struct {
	Code string `json:"code" binding:"required"`
}

func (h *Handlers) VerifyScan(c *gin.Context) {
	userID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req verifyScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	verification, err := h.verifyScan.Execute(c.Request.Context(), application.VerifyScanInput{
		UserID:      userID,
		ScannedCode: req.Code,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidQRCode):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusOK, verification)
}
