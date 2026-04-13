package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	sqlite3 "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	defaultJWTTTLSeconds = 3600
	minPasswordLength    = 8
)

type Ping struct {
	ID      uint   `json:"id" gorm:"primaryKey"`
	Message string `json:"message"`
}

type User struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Email        string    `json:"email" gorm:"uniqueIndex;not null"`
	PasswordHash string    `json:"-" gorm:"not null"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type AuthConfig struct {
	JWTSecret []byte
	JWTTTL    time.Duration
}

type App struct {
	db         *gorm.DB
	authConfig AuthConfig
	now        func() time.Time
	signToken  func(token *jwt.Token) (string, error)
}

type authContextKey string

const authUserIDContextKey authContextKey = "auth_user_id"

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type authResponse struct {
	User  userResponse      `json:"user"`
	Token authTokenResponse `json:"token"`
}

type userResponse struct {
	ID        uint      `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func openDatabase(path string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&Ping{}, &User{}); err != nil {
		return nil, err
	}

	return db, nil
}

func loadAuthConfigFromEnv() (AuthConfig, error) {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		return AuthConfig{}, errors.New("JWT_SECRET is required")
	}

	ttlSeconds, err := loadJWTTTLSecondsFromEnv()
	if err != nil {
		return AuthConfig{}, err
	}

	return AuthConfig{
		JWTSecret: []byte(secret),
		JWTTTL:    time.Duration(ttlSeconds) * time.Second,
	}, nil
}

func loadJWTTTLSecondsFromEnv() (int, error) {
	ttlRaw := strings.TrimSpace(os.Getenv("JWT_TTL"))
	if ttlRaw == "" {
		return defaultJWTTTLSeconds, nil
	}

	parsed, err := strconv.Atoi(ttlRaw)
	if err != nil || parsed <= 0 {
		return 0, errors.New("JWT_TTL must be a positive integer number of seconds")
	}

	return parsed, nil
}

func NewApp(db *gorm.DB, authConfig AuthConfig) *App {
	return &App{
		db:         db,
		authConfig: authConfig,
		now:        time.Now,
		signToken: func(token *jwt.Token) (string, error) {
			return token.SignedString(authConfig.JWTSecret)
		},
	}
}

func (a *App) Router() *gin.Engine {
	r := gin.Default()
	a.registerRoutes(r)
	return r
}

func (a *App) registerRoutes(r *gin.Engine) {
	r.POST("/ping", a.createPing)
	r.GET("/pings", a.listPings)
	r.GET("/hello", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Hello World"})
	})
	r.GET("/time", a.getTime)

	authGroup := r.Group("/auth")
	authGroup.POST("/signup", a.signup)
	authGroup.POST("/login", a.login)

	protected := authGroup.Group("")
	protected.Use(a.authMiddleware())
	protected.GET("/me", a.me)
}

func (a *App) createPing(c *gin.Context) {
	var ping Ping
	if err := c.ShouldBindJSON(&ping); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "invalid ping payload")
		return
	}

	if result := a.db.Create(&ping); result.Error != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to save ping")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Ping registered successfully!"})
}

func (a *App) listPings(c *gin.Context) {
	var pings []Ping
	if result := a.db.Find(&pings); result.Error != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to retrieve pings")
		return
	}

	c.JSON(http.StatusOK, pings)
}

func (a *App) getTime(c *gin.Context) {
	timezone := c.GetHeader("X-Timezone")
	if timezone == "" {
		timezone = "UTC"
	}

	location, err := time.LoadLocation(timezone)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_timezone", "invalid timezone")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"time":     a.now().In(location).Format(time.RFC3339),
		"timezone": timezone,
	})
}

func (a *App) signup(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "email and password are required")
		return
	}

	email, err := normalizeEmail(req.Email)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_email", err.Error())
		return
	}

	if err := validatePassword(req.Password); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_password", err.Error())
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to create user")
		return
	}

	user, err := a.createUser(email, string(passwordHash))
	if err != nil {
		if isDuplicateError(err) {
			writeError(c, http.StatusConflict, "email_exists", "email is already registered")
			return
		}

		writeError(c, http.StatusInternalServerError, "internal_error", "failed to create user")
		return
	}

	token, err := a.generateToken(user.ID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to generate access token")
		return
	}

	c.JSON(http.StatusCreated, authResponse{
		User:  toUserResponse(*user),
		Token: token,
	})
}

func (a *App) login(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "email and password are required")
		return
	}

	email, err := normalizeEmail(req.Email)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}

	user, err := a.findUserByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(c, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
			return
		}

		writeError(c, http.StatusInternalServerError, "internal_error", "failed to login")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeError(c, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}

	token, err := a.generateToken(user.ID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to generate access token")
		return
	}

	c.JSON(http.StatusOK, authResponse{
		User:  toUserResponse(*user),
		Token: token,
	})
}

func (a *App) me(c *gin.Context) {
	id, ok := authUserIDFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	var user User
	if err := a.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(c, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to load user")
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": toUserResponse(user)})
}

func (a *App) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			writeError(c, http.StatusUnauthorized, "missing_token", "authorization token is required")
			c.Abort()
			return
		}

		tokenString, err := parseBearerToken(header)
		if err != nil {
			writeError(c, http.StatusUnauthorized, "invalid_token", err.Error())
			c.Abort()
			return
		}
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return a.authConfig.JWTSecret, nil
		})
		if err != nil || !token.Valid {
			writeError(c, http.StatusUnauthorized, "invalid_token", "invalid or expired token")
			c.Abort()
			return
		}

		id64, err := parseTokenUserID(claims, a.now())
		if err != nil {
			writeError(c, http.StatusUnauthorized, "invalid_token", err.Error())
			c.Abort()
			return
		}

		c.Set(string(authUserIDContextKey), uint(id64))
		c.Next()
	}
}

func authUserIDFromContext(c *gin.Context) (uint, bool) {
	userID, ok := c.Get(string(authUserIDContextKey))
	if !ok {
		return 0, false
	}
	id, ok := userID.(uint)
	if !ok {
		return 0, false
	}
	return id, true
}

func parseBearerToken(header string) (string, error) {
	if header == "" {
		return "", errors.New("authorization token is required")
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", errors.New("invalid authorization header")
	}

	return strings.TrimSpace(parts[1]), nil
}

func parseTokenUserID(claims jwt.MapClaims, now time.Time) (uint64, error) {
	sub, err := claims.GetSubject()
	if err != nil || strings.TrimSpace(sub) == "" {
		return 0, errors.New("invalid token subject")
	}

	expiresAt, err := claims.GetExpirationTime()
	if err != nil || expiresAt == nil || !expiresAt.After(now) {
		return 0, errors.New("invalid or expired token")
	}

	id64, err := strconv.ParseUint(sub, 10, 64)
	if err != nil {
		return 0, errors.New("invalid token subject")
	}

	return id64, nil
}

func (a *App) generateToken(userID uint) (authTokenResponse, error) {
	now := a.now().UTC()
	expiresAt := now.Add(a.authConfig.JWTTTL)

	claims := jwt.MapClaims{
		"sub": strconv.FormatUint(uint64(userID), 10),
		"iat": now.Unix(),
		"exp": expiresAt.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := a.signToken(token)
	if err != nil {
		return authTokenResponse{}, err
	}

	return authTokenResponse{
		AccessToken: signed,
		TokenType:   "Bearer",
		ExpiresIn:   int64(a.authConfig.JWTTTL / time.Second),
	}, nil
}

func (a *App) findUserByEmail(email string) (*User, error) {
	var user User
	if err := a.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (a *App) createUser(email string, passwordHash string) (*User, error) {
	user := User{Email: email, PasswordHash: passwordHash}
	if err := a.db.Create(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func toUserResponse(user User) userResponse {
	return userResponse{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}
}

func normalizeEmail(email string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return "", errors.New("email is required")
	}
	if !strings.Contains(normalized, "@") {
		return "", errors.New("email must be valid")
	}
	return normalized, nil
}

func validatePassword(password string) error {
	if len(password) < minPasswordLength {
		return fmt.Errorf("password must be at least %d characters", minPasswordLength)
	}
	return nil
}

func isDuplicateError(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		return sqliteErr.Code == sqlite3.ErrConstraint && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique
	}

	return strings.Contains(strings.ToLower(err.Error()), "unique constraint")
}

func writeError(c *gin.Context, status int, code string, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}
