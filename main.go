package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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
}

type AuthToken struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"index;not null"`
	Token     string    `json:"token" gorm:"uniqueIndex;not null"`
	ExpiresAt time.Time `json:"expires_at" gorm:"index"`
	CreatedAt time.Time `json:"created_at"`
}

type PasswordResetToken struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	UserID    uint       `json:"user_id" gorm:"index;not null"`
	Token     string     `json:"token" gorm:"uniqueIndex;not null"`
	ExpiresAt time.Time  `json:"expires_at" gorm:"index"`
	UsedAt    *time.Time `json:"used_at" gorm:"index"`
	CreatedAt time.Time  `json:"created_at"`
}

type signupRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type requestResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type resetPasswordRequest struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type userResponse struct {
	ID    uint   `json:"id"`
	Email string `json:"email"`
}

const (
	minPasswordLength  = 8
	authTokenDuration  = 24 * time.Hour
	resetTokenDuration = 1 * time.Hour
	tokenBytes         = 32
)

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func generateToken() (string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func checkPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func authMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization header"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header"})
			c.Abort()
			return
		}

		token := strings.TrimSpace(parts[1])
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization token"})
			c.Abort()
			return
		}

		var authToken AuthToken
		if err := db.Where("token = ?", token).First(&authToken).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		if time.Now().After(authToken.ExpiresAt) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		var user User
		if err := db.First(&user, authToken.UserID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Next()
	}
}

func main() {
	// Initialize the database
	db, err := gorm.Open(sqlite.Open("db/data.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(&Ping{}, &User{}, &AuthToken{}, &PasswordResetToken{})
	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	r := gin.Default()

	// Register routes
	auth := r.Group("/auth")
	auth.POST("/signup", func(c *gin.Context) {
		var req signupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		email := normalizeEmail(req.Email)
		if len(req.Password) < minPasswordLength {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 8 characters"})
			return
		}

		var existing User
		if err := db.Where("email = ?", email).First(&existing).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
			return
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing user"})
			return
		}

		passwordHash, err := hashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}

		user := User{Email: email, PasswordHash: passwordHash}
		if err := db.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}

		c.JSON(http.StatusCreated, userResponse{ID: user.ID, Email: user.Email})
	})

	auth.POST("/login", func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		email := normalizeEmail(req.Email)
		var user User
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
			return
		}

		if !checkPassword(req.Password, user.PasswordHash) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
			return
		}

		token, err := generateToken()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		expiresAt := time.Now().Add(authTokenDuration)
		authToken := AuthToken{UserID: user.ID, Token: token, ExpiresAt: expiresAt}
		if err := db.Create(&authToken).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create auth token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token":      token,
			"expires_at": expiresAt.Format(time.RFC3339),
			"user":       userResponse{ID: user.ID, Email: user.Email},
		})
	})

	auth.GET("/me", authMiddleware(db), func(c *gin.Context) {
		value, ok := c.Get("user")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		user, ok := value.(User)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		c.JSON(http.StatusOK, userResponse{ID: user.ID, Email: user.Email})
	})

	auth.POST("/request-password-reset", func(c *gin.Context) {
		var req requestResetRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		email := normalizeEmail(req.Email)
		var user User
		if err := db.Where("email = ?", email).First(&user).Error; err == nil {
			token, err := generateToken()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate reset token"})
				return
			}

			resetToken := PasswordResetToken{
				UserID:    user.ID,
				Token:     token,
				ExpiresAt: time.Now().Add(resetTokenDuration),
			}
			if err := db.Create(&resetToken).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create reset token"})
				return
			}

			log.Printf("SMTP stub: password reset token for %s: %s", user.Email, token)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create reset token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "If that email exists, a reset token has been sent"})
	})

	auth.POST("/reset-password", func(c *gin.Context) {
		var req resetPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if len(req.Password) < minPasswordLength {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 8 characters"})
			return
		}

		var resetToken PasswordResetToken
		if err := db.Where("token = ? AND used_at IS NULL AND expires_at > ?", strings.TrimSpace(req.Token), time.Now()).First(&resetToken).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired reset token"})
			return
		}

		passwordHash, err := hashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&User{}).Where("id = ?", resetToken.UserID).Update("password_hash", passwordHash).Error; err != nil {
				return err
			}

			now := time.Now()
			if err := tx.Model(&PasswordResetToken{}).Where("id = ?", resetToken.ID).Update("used_at", &now).Error; err != nil {
				return err
			}

			if err := tx.Where("user_id = ?", resetToken.UserID).Delete(&AuthToken{}).Error; err != nil {
				return err
			}

			return nil
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Password reset successful"})
	})

	r.POST("/ping", func(c *gin.Context) {
		var ping Ping
		if err := c.ShouldBindJSON(&ping); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Save the ping to the database
		if result := db.Create(&ping); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save ping"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Ping registered successfully!"})
	})

	r.GET("/pings", func(c *gin.Context) {
		var pings []Ping

		// Retrieve all pings from the database
		if result := db.Find(&pings); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve pings"})
			return
		}

		c.JSON(http.StatusOK, pings)
	})

	r.GET("/hello", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Hello World"})
	})

	r.GET("/time", func(c *gin.Context) {
		timezone := c.GetHeader("X-Timezone")
		if timezone == "" {
			timezone = "UTC"
		}

		location, err := time.LoadLocation(timezone)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timezone"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"time":     time.Now().In(location).Format(time.RFC3339),
			"timezone": timezone,
		})
	})

	// Start the server
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
