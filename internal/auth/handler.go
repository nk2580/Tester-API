package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nk2580/Tester-API/internal/data"
	"github.com/nk2580/Tester-API/internal/httputil"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Handler manages authentication-related HTTP endpoints.
type Handler struct {
    users  data.UserStore
    tokens TokenService
}

// NewHandler constructs a Handler.
func NewHandler(users data.UserStore, tokens TokenService) *Handler {
    return &Handler{users: users, tokens: tokens}
}

type authRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

type userResponse struct {
    ID        uint   `json:"id"`
    Email     string `json:"email"`
    CreatedAt string `json:"created_at"`
}

type authResponse struct {
    User  userResponse `json:"user"`
    Token Token        `json:"token"`
}

// Signup creates a new user account and returns an access token.
func (h *Handler) Signup(c *gin.Context) {
    var req authRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        httputil.WriteError(c, http.StatusBadRequest, "invalid_request", "email and password are required")
        return
    }

    email, err := normalizeEmail(req.Email)
    if err != nil {
        httputil.WriteError(c, http.StatusBadRequest, "invalid_email", err.Error())
        return
    }

    if err := validatePassword(req.Password); err != nil {
        httputil.WriteError(c, http.StatusBadRequest, "invalid_password", err.Error())
        return
    }

    passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    if err != nil {
        httputil.WriteError(c, http.StatusInternalServerError, "internal_error", "failed to create user")
        return
    }

    user := &data.User{Email: email, PasswordHash: string(passwordHash)}
    if err := h.users.CreateUser(c.Request.Context(), user); err != nil {
        if data.IsDuplicateError(err) {
            httputil.WriteError(c, http.StatusConflict, "email_exists", "email is already registered")
            return
        }
        httputil.WriteError(c, http.StatusInternalServerError, "internal_error", "failed to create user")
        return
    }

    token, err := h.tokens.Generate(user.ID)
    if err != nil {
        httputil.WriteError(c, http.StatusInternalServerError, "internal_error", "failed to generate access token")
        return
    }

    c.JSON(http.StatusCreated, authResponse{
        User:  toUserResponse(user),
        Token: token,
    })
}

// Login authenticates the provided credentials and returns an access token.
func (h *Handler) Login(c *gin.Context) {
    var req authRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        httputil.WriteError(c, http.StatusBadRequest, "invalid_request", "email and password are required")
        return
    }

    email, err := normalizeEmail(req.Email)
    if err != nil {
        httputil.WriteError(c, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
        return
    }

    user, err := h.users.FindByEmail(c.Request.Context(), email)
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            httputil.WriteError(c, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
            return
        }
        httputil.WriteError(c, http.StatusInternalServerError, "internal_error", "failed to login")
        return
    }

    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
        httputil.WriteError(c, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
        return
    }

    token, err := h.tokens.Generate(user.ID)
    if err != nil {
        httputil.WriteError(c, http.StatusInternalServerError, "internal_error", "failed to generate access token")
        return
    }

    c.JSON(http.StatusOK, authResponse{
        User:  toUserResponse(user),
        Token: token,
    })
}

// Me returns the authenticated user's profile.
func (h *Handler) Me(c *gin.Context) {
    rawID, exists := c.Get(contextUserIDKey)
    if !exists {
        httputil.WriteError(c, http.StatusUnauthorized, "unauthorized", "authentication required")
        return
    }

    id, ok := rawID.(uint)
    if !ok {
        httputil.WriteError(c, http.StatusUnauthorized, "unauthorized", "authentication required")
        return
    }

    user, err := h.users.FindByID(c.Request.Context(), id)
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            httputil.WriteError(c, http.StatusUnauthorized, "unauthorized", "authentication required")
            return
        }
        httputil.WriteError(c, http.StatusInternalServerError, "internal_error", "failed to load user")
        return
    }

    c.JSON(http.StatusOK, gin.H{"user": toUserResponse(user)})
}

func toUserResponse(user *data.User) userResponse {
    return userResponse{
        ID:        user.ID,
        Email:     user.Email,
        CreatedAt: user.CreatedAt.Format(time.RFC3339),
    }
}
