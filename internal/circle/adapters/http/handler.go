package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/bLorax/khatere-backend/internal/circle/application"
	"github.com/bLorax/khatere-backend/internal/circle/domain"
	userDomain "github.com/bLorax/khatere-backend/internal/user/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UserFinder is intentionally small so the circle HTTP adapter doesn't
// depend on the concrete PostgreSQL user repository.
//
// UserRepository will satisfy this interface once FindByHandle is added
// in the user-domain touch-up.
type UserFinder interface {
	FindByAccountID(ctx context.Context, accountID uuid.UUID) (*userDomain.User, error)
	FindByHandle(ctx context.Context, handle string) (*userDomain.User, error)
}

type Handlers struct {
	sendRequest     *application.SendConnectionRequestUseCase
	acceptRequest   *application.AcceptConnectionRequestUseCase
	declineRequest  *application.DeclineConnectionRequestUseCase
	severConnection *application.SeverConnectionUseCase
	blockUser       *application.BlockUserUseCase
	unblockUser     *application.UnblockUserUseCase
	listCircle      *application.ListCircleUseCase
	listPending     *application.ListPendingRequestsUseCase
	blocks          domain.BlockRepository
	users           UserFinder
}

func NewHandlers(
	sendRequest *application.SendConnectionRequestUseCase,
	acceptRequest *application.AcceptConnectionRequestUseCase,
	declineRequest *application.DeclineConnectionRequestUseCase,
	severConnection *application.SeverConnectionUseCase,
	blockUser *application.BlockUserUseCase,
	unblockUser *application.UnblockUserUseCase,
	listCircle *application.ListCircleUseCase,
	listPending *application.ListPendingRequestsUseCase,
	blocks domain.BlockRepository,
	users UserFinder,
) *Handlers {
	return &Handlers{
		sendRequest:     sendRequest,
		acceptRequest:   acceptRequest,
		declineRequest:  declineRequest,
		severConnection: severConnection,
		blockUser:       blockUser,
		unblockUser:     unblockUser,
		listCircle:      listCircle,
		listPending:     listPending,
		blocks:          blocks,
		users:           users,
	}
}

func (h *Handlers) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/requests", h.SendConnectionRequest)
	r.POST("/requests/:id/accept", h.AcceptConnectionRequest)
	r.POST("/requests/:id/decline", h.DeclineConnectionRequest)

	r.GET("/requests/incoming", h.ListIncomingRequests)
	r.GET("/requests/outgoing", h.ListOutgoingRequests)

	r.GET("", h.ListCircle)
	r.DELETE("/connections/:id", h.SeverConnection)

	r.POST("/blocks", h.BlockUser)
	r.DELETE("/blocks/:userID", h.UnblockUser)
	r.GET("/blocks", h.ListBlocks)
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

type sendConnectionRequest struct {
	UserID *string `json:"user_id"`
	Handle *string `json:"handle"`
}

func (h *Handlers) SendConnectionRequest(c *gin.Context) {
	accountID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req sendConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if (req.UserID == nil) == (req.Handle == nil) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "exactly one of user_id or handle is required",
		})
		return
	}

	var targetID uuid.UUID

	if req.UserID != nil {
		targetID, err = uuid.Parse(*req.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
			return
		}
	} else {
		target, err := h.users.FindByHandle(c.Request.Context(), *req.Handle)
		if err != nil {
			if errors.Is(err, userDomain.ErrUserNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		targetID = target.AccountID
	}

	connection, err := h.sendRequest.Execute(
		c.Request.Context(),
		application.SendConnectionRequestInput{
			RequesterID: accountID,
			AddresseeID: targetID,
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCannotConnectSelf):
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot connect to yourself"})
		case errors.Is(err, domain.ErrBlocked):
			c.JSON(http.StatusForbidden, gin.H{"error": "user is blocked"})
		case errors.Is(err, domain.ErrAlreadyConnected):
			c.JSON(http.StatusConflict, gin.H{"error": "already connected"})
		case errors.Is(err, domain.ErrRequestAlreadySent):
			c.JSON(http.StatusConflict, gin.H{"error": "connection request already sent"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusCreated, connection)
}

func (h *Handlers) AcceptConnectionRequest(c *gin.Context) {
	accountID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request id"})
		return
	}

	connection, err := h.acceptRequest.Execute(
		c.Request.Context(),
		application.AcceptConnectionRequestInput{
			RequestID:   requestID,
			AddresseeID: accountID,
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrRequestNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "connection request not found"})
		case errors.Is(err, domain.ErrBlocked):
			c.JSON(http.StatusForbidden, gin.H{"error": "user is blocked"})
		case errors.Is(err, domain.ErrAlreadyConnected):
			c.JSON(http.StatusConflict, gin.H{"error": "already connected"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusOK, connection)
}

func (h *Handlers) DeclineConnectionRequest(c *gin.Context) {
	accountID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request id"})
		return
	}

	err = h.declineRequest.Execute(
		c.Request.Context(),
		application.DeclineConnectionRequestInput{
			RequestID:   requestID,
			AddresseeID: accountID,
		},
	)
	if err != nil {
		if errors.Is(err, domain.ErrRequestNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "connection request not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handlers) ListIncomingRequests(c *gin.Context) {
	accountID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	pending, err := h.listPending.Execute(
		c.Request.Context(),
		accountID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, pending.Incoming)
}

func (h *Handlers) ListOutgoingRequests(c *gin.Context) {
	accountID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	pending, err := h.listPending.Execute(
		c.Request.Context(),
		accountID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, pending.Outgoing)
}

func (h *Handlers) ListCircle(c *gin.Context) {
	accountID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	connections, err := h.listCircle.Execute(
		c.Request.Context(),
		accountID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, connections)
}

func (h *Handlers) SeverConnection(c *gin.Context) {
	accountID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	connectionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid connection id"})
		return
	}

	err = h.severConnection.Execute(
		c.Request.Context(),
		application.SeverConnectionInput{
			ConnectionID: connectionID,
			UserID:       accountID,
		},
	)
	if err != nil {
		if errors.Is(err, domain.ErrConnectionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.Status(http.StatusNoContent)
}

type blockUserRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

func (h *Handlers) BlockUser(c *gin.Context) {
	accountID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req blockUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	blockedID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	block, err := h.blockUser.Execute(
		c.Request.Context(),
		application.BlockUserInput{
			BlockerID: accountID,
			BlockedID: blockedID,
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCannotBlockSelf):
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot block yourself"})
		case errors.Is(err, domain.ErrAlreadyBlocked):
			c.JSON(http.StatusConflict, gin.H{"error": "user already blocked"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusCreated, block)
}

func (h *Handlers) UnblockUser(c *gin.Context) {
	accountID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	blockedID, err := uuid.Parse(c.Param("userID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	err = h.unblockUser.Execute(
		c.Request.Context(),
		application.UnblockUserInput{
			BlockerID: accountID,
			BlockedID: blockedID,
		},
	)
	if err != nil {
		if errors.Is(err, domain.ErrBlockNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "block not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handlers) ListBlocks(c *gin.Context) {
	accountID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	blocks, err := h.blocks.List(c.Request.Context(), accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, blocks)
}
