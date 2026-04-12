package app

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nk2580/Tester-API/internal/auth"
	"github.com/nk2580/Tester-API/internal/config"
	"github.com/nk2580/Tester-API/internal/models"
	"gorm.io/gorm"
)

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

type Application struct {
	db           *gorm.DB
	cfg          config.Config
	mailer       auth.Mailer
	rateLimiters map[string]*auth.FixedWindowRateLimiter
	now          func() time.Time
}

func New(db *gorm.DB, cfg config.Config, mailer auth.Mailer, nowFn func() time.Time) *Application {
	if nowFn == nil {
		nowFn = time.Now
	}
	return &Application{
		db:     db,
		cfg:    cfg,
		mailer: mailer,
		now:    nowFn,
		rateLimiters: map[string]*auth.FixedWindowRateLimiter{
			"register":            auth.NewFixedWindowRateLimiter(20, time.Minute),
			"verify":              auth.NewFixedWindowRateLimiter(30, time.Minute),
			"resend_verification": auth.NewFixedWindowRateLimiter(10, time.Minute),
			"login":               auth.NewFixedWindowRateLimiter(30, time.Minute),
			"forgot_password":     auth.NewFixedWindowRateLimiter(15, time.Minute),
			"reset_password":      auth.NewFixedWindowRateLimiter(20, time.Minute),
		},
	}
}

func (a *Application) RegisterRoutes(r *gin.Engine) {
	r.POST("/ping", a.createPing)
	r.GET("/pings", a.listPings)
	r.GET("/hello", a.hello)
	r.GET("/time", a.currentTime)

	authRoutes := r.Group("/auth")
	authRoutes.POST("/register", a.rateLimit("register"), a.register)
	authRoutes.POST("/verify-email", a.rateLimit("verify"), a.verifyEmail)
	authRoutes.POST("/resend-verification", a.rateLimit("resend_verification"), a.resendVerification)
	authRoutes.POST("/login", a.rateLimit("login"), a.login)
	authRoutes.POST("/logout", a.logout)
	authRoutes.POST("/forgot-password", a.rateLimit("forgot_password"), a.forgotPassword)
	authRoutes.POST("/reset-password", a.rateLimit("reset_password"), a.resetPassword)
	authRoutes.GET("/me", a.authRequired(), a.me)
}

func (a *Application) createPing(c *gin.Context) {
	var ping models.Ping
	if err := c.ShouldBindJSON(&ping); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if result := a.db.Create(&ping); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save ping"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Ping registered successfully!"})
}

func (a *Application) listPings(c *gin.Context) {
	var pings []models.Ping
	if result := a.db.Find(&pings); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve pings"})
		return
	}
	c.JSON(http.StatusOK, pings)
}

func (a *Application) hello(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Hello World"})
}

func (a *Application) currentTime(c *gin.Context) {
	timezone := c.GetHeader("X-Timezone")
	if timezone == "" {
		timezone = "UTC"
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid timezone"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"time": a.now().In(location).Format(time.RFC3339), "timezone": timezone})
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *Application) register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	email := normalizeEmail(req.Email)
	if !isValidEmail(email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}
	if err := validatePasswordPolicy(req.Password, a.cfg.PasswordMinLength); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := a.now()
	var user models.User
	lookup := a.db.Where("email = ?", email).First(&user)
	if lookup.Error == nil {
		if user.State != models.UserStateActive {
			a.maybeResendVerificationToken(c.ClientIP(), &user, now)
		}
		c.JSON(http.StatusAccepted, gin.H{"message": "If the account is eligible, a verification email has been sent"})
		return
	}
	if !errors.Is(lookup.Error, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register"})
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register"})
		return
	}

	newUser := models.User{Email: email, PasswordHash: passwordHash, State: models.UserStatePendingVerification}
	if err := a.db.Create(&newUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register"})
		return
	}

	a.issueVerificationToken(c.ClientIP(), &newUser, now)
	c.JSON(http.StatusAccepted, gin.H{"message": "If the account is eligible, a verification email has been sent"})
}

type verifyEmailRequest struct {
	Token string `json:"token"`
}

func (a *Application) verifyEmail(c *gin.Context) {
	var req verifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Token) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token"})
		return
	}

	now := a.now()
	tokenHash := auth.HashToken(strings.TrimSpace(req.Token))
	var token models.VerificationToken
	if err := a.db.Where("token_hash = ?", tokenHash).First(&token).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired token"})
		return
	}
	if token.UsedAt != nil || now.After(token.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired token"})
		return
	}

	var user models.User
	if err := a.db.First(&user, token.UserID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired token"})
		return
	}

	verifiedAt := now
	token.UsedAt = &verifiedAt
	user.State = models.UserStateActive
	user.EmailVerifiedAt = &verifiedAt
	user.LockedUntil = nil
	user.FailedLoginAttempts = 0
	if err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&token).Error; err != nil {
			return err
		}
		if err := tx.Save(&user).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ? AND used_at IS NULL AND id <> ?", user.ID, token.ID).
			Delete(&models.VerificationToken{}).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify email"})
		return
	}

	a.logAudit(&user.ID, "email_verified", c.ClientIP(), "")
	c.JSON(http.StatusOK, gin.H{"message": "email verified"})
}

type resendVerificationRequest struct {
	Email string `json:"email"`
}

func (a *Application) resendVerification(c *gin.Context) {
	var req resendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	email := normalizeEmail(req.Email)

	now := a.now()
	var user models.User
	if err := a.db.Where("email = ?", email).First(&user).Error; err == nil && user.State != models.UserStateActive {
		a.maybeResendVerificationToken(c.ClientIP(), &user, now)
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "If the account is eligible, a verification email has been sent"})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *Application) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	now := a.now()
	email := normalizeEmail(req.Email)
	var user models.User
	err := a.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		a.handleInvalidLogin(c, nil, "user not found")
		return
	}

	if user.LockedUntil != nil && now.Before(*user.LockedUntil) {
		a.logAudit(&user.ID, "login_blocked_locked", c.ClientIP(), "account locked")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	if user.State != models.UserStateActive {
		a.handleInvalidLogin(c, &user, "account not active")
		return
	}

	ok, verifyErr := auth.VerifyPassword(user.PasswordHash, req.Password)
	if verifyErr != nil || !ok {
		a.handleInvalidLogin(c, &user, "invalid password")
		return
	}

	if err := a.resetLoginFailures(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to login"})
		return
	}

	token, err := auth.GenerateToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to login"})
		return
	}
	tokenHash := auth.HashToken(token)

	session := models.Session{UserID: user.ID, TokenHash: tokenHash, ExpiresAt: now.Add(a.cfg.SessionTTL)}
	if err := a.db.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to login"})
		return
	}

	a.setSessionCookie(c, token, session.ExpiresAt)
	a.logAudit(&user.ID, "login_success", c.ClientIP(), "")
	c.JSON(http.StatusOK, gin.H{"message": "login successful"})
}

func (a *Application) logout(c *gin.Context) {
	token, err := c.Cookie(a.cfg.SessionCookieName)
	if err == nil && strings.TrimSpace(token) != "" {
		hash := auth.HashToken(token)
		now := a.now()
		a.db.Model(&models.Session{}).
			Where("token_hash = ? AND revoked_at IS NULL", hash).
			Update("revoked_at", now)
	}

	a.clearSessionCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

func (a *Application) forgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	email := normalizeEmail(req.Email)
	now := a.now()
	var user models.User
	if err := a.db.Where("email = ?", email).First(&user).Error; err == nil && user.State == models.UserStateActive {
		token, tokenErr := auth.GenerateToken()
		if tokenErr == nil {
			hash := auth.HashToken(token)
			resetToken := models.PasswordResetToken{UserID: user.ID, TokenHash: hash, ExpiresAt: now.Add(a.cfg.PasswordResetTokenTTL)}
			if err := a.db.Create(&resetToken).Error; err == nil {
				a.logAudit(&user.ID, "password_reset_requested", c.ClientIP(), "")
				_ = a.mailer.SendPasswordResetEmail(user.Email, token)
			}
		}
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "If an account exists, password reset instructions have been sent"})
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (a *Application) resetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	if strings.TrimSpace(req.Token) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token"})
		return
	}
	if err := validatePasswordPolicy(req.NewPassword, a.cfg.PasswordMinLength); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := a.now()
	hash := auth.HashToken(strings.TrimSpace(req.Token))
	var resetToken models.PasswordResetToken
	if err := a.db.Where("token_hash = ?", hash).First(&resetToken).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired token"})
		return
	}
	if resetToken.UsedAt != nil || now.After(resetToken.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired token"})
		return
	}

	var user models.User
	if err := a.db.First(&user, resetToken.UserID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired token"})
		return
	}

	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset password"})
		return
	}

	nowCopy := now
	if err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.PasswordResetToken{}).
			Where("user_id = ? AND used_at IS NULL", user.ID).
			Updates(map[string]any{"used_at": nowCopy}).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.User{}).
			Where("id = ?", user.ID).
			Updates(map[string]any{
				"password_hash":         newHash,
				"failed_login_attempts": 0,
				"locked_until":          nil,
				"state":                 models.UserStateActive,
			}).Error; err != nil {
			return err
		}

		return tx.Model(&models.Session{}).Where("user_id = ? AND revoked_at IS NULL", user.ID).
			Updates(map[string]any{"revoked_at": nowCopy}).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset password"})
		return
	}

	a.logAudit(&user.ID, "password_reset_completed", c.ClientIP(), "")
	_ = a.mailer.SendPasswordResetConfirmationEmail(user.Email)
	c.JSON(http.StatusOK, gin.H{"message": "password reset successful"})
}

func (a *Application) me(c *gin.Context) {
	userValue, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	user := userValue.(models.User)
	c.JSON(http.StatusOK, gin.H{"id": user.ID, "email": user.Email, "state": user.State})
}

func (a *Application) authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(a.cfg.SessionCookieName)
		if err != nil || strings.TrimSpace(token) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
			return
		}

		hash := auth.HashToken(token)
		now := a.now()

		var session models.Session
		if err := a.db.Where("token_hash = ?", hash).First(&session).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
			return
		}
		if session.RevokedAt != nil || now.After(session.ExpiresAt) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
			return
		}

		var user models.User
		if err := a.db.First(&user, session.UserID).Error; err != nil || user.State != models.UserStateActive {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
			return
		}

		c.Set("user", user)
		c.Next()
	}
}

func (a *Application) rateLimit(name string) gin.HandlerFunc {
	limiter, ok := a.rateLimiters[name]
	if !ok {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		key := fmt.Sprintf("%s:%s", name, c.ClientIP())
		if !limiter.Allow(key, a.now()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}
		c.Next()
	}
}

func (a *Application) issueVerificationToken(ip string, user *models.User, now time.Time) {
	token, err := auth.GenerateToken()
	if err != nil {
		return
	}
	verification := models.VerificationToken{UserID: user.ID, TokenHash: auth.HashToken(token), ExpiresAt: now.Add(a.cfg.VerificationTokenTTL)}
	if err := a.db.Create(&verification).Error; err != nil {
		return
	}
	user.LastVerificationSentAt = &now
	a.db.Model(user).Update("last_verification_sent_at", now)
	a.logAudit(&user.ID, "verification_sent", ip, "")
	_ = a.mailer.SendVerificationEmail(user.Email, token)
}

func (a *Application) maybeResendVerificationToken(ip string, user *models.User, now time.Time) {
	if user.LastVerificationSentAt != nil && now.Sub(*user.LastVerificationSentAt) < a.cfg.ResendVerificationCooldown {
		return
	}
	a.issueVerificationToken(ip, user, now)
}

func (a *Application) handleInvalidLogin(c *gin.Context, user *models.User, reason string) {
	if user != nil {
		now := a.now()
		updates := map[string]any{"failed_login_attempts": user.FailedLoginAttempts + 1}
		attempts := user.FailedLoginAttempts + 1
		if attempts >= a.cfg.MaxFailedLoginAttempts {
			lockUntil := now.Add(a.cfg.LockoutDuration)
			updates["locked_until"] = lockUntil
			updates["state"] = models.UserStateLocked
		}
		a.db.Model(&models.User{}).Where("id = ?", user.ID).Updates(updates)
		a.logAudit(&user.ID, "login_failed", c.ClientIP(), reason)
	} else {
		a.logAudit(nil, "login_failed", c.ClientIP(), reason)
	}
	c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
}

func (a *Application) resetLoginFailures(user *models.User) error {
	updates := map[string]any{"failed_login_attempts": 0, "locked_until": nil}
	if user.State == models.UserStateLocked {
		updates["state"] = models.UserStateActive
	}
	return a.db.Model(&models.User{}).Where("id = ?", user.ID).Updates(updates).Error
}

func (a *Application) setSessionCookie(c *gin.Context, token string, expiresAt time.Time) {
	cookie := &http.Cookie{
		Name:     a.cfg.SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.cfg.CookieSecure,
		SameSite: http.SameSiteStrictMode,
		Expires:  expiresAt,
	}
	if strings.TrimSpace(a.cfg.CookieDomain) != "" {
		cookie.Domain = a.cfg.CookieDomain
	}
	http.SetCookie(c.Writer, cookie)
}

func (a *Application) clearSessionCookie(c *gin.Context) {
	cookie := &http.Cookie{
		Name:     a.cfg.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   a.cfg.CookieSecure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	}
	if strings.TrimSpace(a.cfg.CookieDomain) != "" {
		cookie.Domain = a.cfg.CookieDomain
	}
	http.SetCookie(c.Writer, cookie)
}

func (a *Application) logAudit(userID *uint, eventType, ip, details string) {
	event := models.AuditEvent{UserID: userID, EventType: eventType, IP: ip, Details: details}
	a.db.Create(&event)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func validatePasswordPolicy(password string, minLen int) error {
	if len(password) < minLen {
		return fmt.Errorf("password must be at least %d characters", minLen)
	}
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasDigit = true
		default:
			hasSpecial = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
		return errors.New("password must include uppercase, lowercase, number, and special character")
	}
	return nil
}
