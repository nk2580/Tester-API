package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nk2580/Tester-API/internal/api"
	"github.com/nk2580/Tester-API/internal/models"
	"github.com/nk2580/Tester-API/internal/services"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type registerRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type resetRequest struct {
	Email string `json:"email" binding:"required"`
}

type resetConfirmRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ValidationError(c, "Invalid registration payload")
		return
	}
	user, err := h.authService.Register(req.Email, req.Password, c.ClientIP())
	if err != nil {
		if isUniqueConstraint(err) {
			api.Error(c, http.StatusConflict, "registration_failed", "Unable to register with provided credentials")
			return
		}
		api.ValidationError(c, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": user.ID, "email": user.Email, "created_at": user.CreatedAt})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ValidationError(c, "Invalid login payload")
		return
	}
	token, expiresAt, user, err := h.authService.Login(req.Email, req.Password, c.ClientIP())
	if err != nil {
		switch err {
		case services.ErrTooManyAttempts:
			api.Error(c, http.StatusTooManyRequests, "login_rate_limited", "Too many failed login attempts")
		case services.ErrAccountUnverified:
			api.Forbidden(c, "Account not verified")
		default:
			api.Error(c, http.StatusUnauthorized, "auth_failed", "Invalid email or password")
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   int(time.Until(expiresAt).Seconds()),
		"user":         gin.H{"id": user.ID, "email": user.Email, "last_login_at": user.LastLoginAt},
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	userValue, ok := c.Get("auth_user")
	if !ok {
		api.Unauthorized(c)
		return
	}
	sessionValue, ok := c.Get("auth_session_id")
	if !ok {
		api.Unauthorized(c)
		return
	}

	user := userValue.(*models.User)
	sessionID := sessionValue.(uint)
	if err := h.authService.Logout(sessionID, user.ID); err != nil {
		api.Internal(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}

func (h *AuthHandler) PasswordResetRequest(c *gin.Context) {
	var req resetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ValidationError(c, "Invalid password reset request")
		return
	}
	if err := h.authService.RequestPasswordReset(req.Email, c.ClientIP()); err != nil {
		api.Internal(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "If an account exists for that email, a reset link has been sent"})
}

func (h *AuthHandler) PasswordResetConfirm(c *gin.Context) {
	var req resetConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ValidationError(c, "Invalid password reset payload")
		return
	}
	if err := h.authService.ConfirmPasswordReset(req.Token, req.NewPassword); err != nil {
		switch err {
		case services.ErrResetTokenInvalid, services.ErrResetTokenConsumed:
			api.Error(c, http.StatusBadRequest, "invalid_token", "Invalid or expired reset token")
		default:
			api.ValidationError(c, err.Error())
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userValue, ok := c.Get("auth_user")
	if !ok {
		api.Unauthorized(c)
		return
	}
	user := userValue.(*models.User)
	c.JSON(http.StatusOK, gin.H{"id": user.ID, "email": user.Email, "is_verified": user.IsVerified})
}

func isUniqueConstraint(err error) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), "UNIQUE constraint failed")
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
