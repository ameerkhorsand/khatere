package http

import (
	"errors"
	"net/http"

	"github.com/bLorax/khatere-backend/internal/auth/application"
	"github.com/bLorax/khatere-backend/internal/auth/domain"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	register *application.RegisterUseCase
	login    *application.LoginUseCase
	refresh  *application.RefreshUseCase
}

func NewHandlers(
	register *application.RegisterUseCase,
	login *application.LoginUseCase,
	refresh *application.RefreshUseCase,
) *Handlers {
	return &Handlers{register, login, refresh}
}

func (h *Handlers) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)
	r.POST("/refresh", h.Refresh)
}

type registerRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	AccountType string `json:"account_type" binding:"required"`
}

func (h *Handlers) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	account, err := h.register.Execute(c.Request.Context(), application.RegisterInput{
		Email:       req.Email,
		Password:    req.Password,
		AccountType: domain.AccountType(req.AccountType),
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrEmailAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		case errors.Is(err, domain.ErrInvalidAccountType):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account type"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": account.ID, "email": account.Email})
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *Handlers) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	out, err := h.login.Execute(c.Request.Context(), application.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  out.AccessToken,
		"refresh_token": out.RefreshToken,
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *Handlers) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	out, err := h.refresh.Execute(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  out.AccessToken,
		"refresh_token": out.RefreshToken,
	})
}
