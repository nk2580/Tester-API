package services

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/nk2580/Tester-API/internal/config"
	"github.com/nk2580/Tester-API/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrAuthFailed         = errors.New("invalid email or password")
	ErrAccountUnverified  = errors.New("account not verified")
	ErrTooManyAttempts    = errors.New("too many login attempts")
	ErrResetTokenInvalid  = errors.New("invalid or expired reset token")
	ErrResetTokenConsumed = errors.New("reset token already used")
)

type AuthService struct {
	db      *gorm.DB
	cfg     config.Config
	email   EmailSender
	auditor *AuditService
}

func NewAuthService(db *gorm.DB, cfg config.Config, email EmailSender, auditor *AuditService) *AuthService {
	return &AuthService{db: db, cfg: cfg, email: email, auditor: auditor}
}

func (s *AuthService) Register(email, password, ip string) (*models.User, error) {
	email = NormalizeEmail(email)
	if err := ValidateEmail(email); err != nil {
		return nil, err
	}
	if err := ValidatePasswordPolicy(password); err != nil {
		return nil, err
	}

	hash, err := HashPassword(password, s.cfg.BcryptCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{Email: email, PasswordHash: hash, IsVerified: true}
	if err := s.db.Create(user).Error; err != nil {
		return nil, err
	}

	s.auditor.Record("auth.register", &user.ID, map[string]any{"ip": ip, "email": user.Email})
	return user, nil
}

func (s *AuthService) Login(email, password, ip string) (string, time.Time, *models.User, error) {
	email = NormalizeEmail(email)
	if s.isLocked(email, ip) {
		return "", time.Time{}, nil, ErrTooManyAttempts
	}

	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		s.recordFailedAttempt(email, ip)
		return "", time.Time{}, nil, ErrAuthFailed
	}
	if !VerifyPassword(user.PasswordHash, password) {
		s.recordFailedAttempt(email, ip)
		return "", time.Time{}, nil, ErrAuthFailed
	}
	if !user.IsVerified {
		return "", time.Time{}, nil, ErrAccountUnverified
	}

	s.clearFailedAttempt(email, ip)
	now := time.Now().UTC()
	expires := now.Add(s.cfg.AccessTokenTTL)
	token, err := GenerateToken()
	if err != nil {
		return "", time.Time{}, nil, err
	}

	sess := &models.Session{UserID: user.ID, TokenHash: HashToken(token), ExpiresAt: expires}
	if err := s.db.Create(sess).Error; err != nil {
		return "", time.Time{}, nil, err
	}
	_ = s.db.Model(&user).Update("last_login_at", now).Error

	s.auditor.Record("auth.login", &user.ID, map[string]any{"ip": ip})
	return token, expires, &user, nil
}

func (s *AuthService) Authenticate(token string) (*models.User, *models.Session, error) {
	if token == "" {
		return nil, nil, ErrAuthFailed
	}
	tokenHash := HashToken(token)
	var sess models.Session
	if err := s.db.Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", tokenHash, time.Now().UTC()).First(&sess).Error; err != nil {
		return nil, nil, ErrAuthFailed
	}
	var user models.User
	if err := s.db.First(&user, sess.UserID).Error; err != nil {
		return nil, nil, ErrAuthFailed
	}
	return &user, &sess, nil
}

func (s *AuthService) Logout(sessionID uint, userID uint) error {
	now := time.Now().UTC()
	if err := s.db.Model(&models.Session{}).Where("id = ? AND user_id = ? AND revoked_at IS NULL", sessionID, userID).Update("revoked_at", now).Error; err != nil {
		return err
	}
	s.auditor.Record("auth.logout", &userID, map[string]any{})
	return nil
}

func (s *AuthService) RequestPasswordReset(email, ip string) error {
	email = NormalizeEmail(email)
	if err := ValidateEmail(email); err != nil {
		return nil
	}

	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil
	}

	token, err := GenerateToken()
	if err != nil {
		return err
	}
	tokenHash := HashToken(token)
	expires := time.Now().UTC().Add(s.cfg.PasswordResetTokenTTL)

	row := models.PasswordResetToken{UserID: user.ID, TokenHash: tokenHash, ExpiresAt: expires}
	if err := s.db.Create(&row).Error; err != nil {
		return err
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.cfg.BaseURL, url.QueryEscape(token))
	if err := s.email.SendPasswordResetEmail(user.Email, resetURL); err != nil {
		return err
	}

	s.auditor.Record("auth.password_reset.request", &user.ID, map[string]any{"ip": ip})
	return nil
}

func (s *AuthService) ConfirmPasswordReset(token, newPassword string) error {
	if err := ValidatePasswordPolicy(newPassword); err != nil {
		return err
	}
	tokenHash := HashToken(token)
	now := time.Now().UTC()

	tx := s.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	var resetToken models.PasswordResetToken
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("token_hash = ?", tokenHash).First(&resetToken).Error; err != nil {
		tx.Rollback()
		return ErrResetTokenInvalid
	}
	if resetToken.UsedAt != nil {
		tx.Rollback()
		return ErrResetTokenConsumed
	}
	if now.After(resetToken.ExpiresAt) {
		tx.Rollback()
		return ErrResetTokenInvalid
	}

	result := tx.Model(&models.PasswordResetToken{}).Where("id = ? AND used_at IS NULL", resetToken.ID).Update("used_at", now)
	if result.Error != nil {
		tx.Rollback()
		return result.Error
	}
	if result.RowsAffected != 1 {
		tx.Rollback()
		return ErrResetTokenConsumed
	}

	pwdHash, err := HashPassword(newPassword, s.cfg.BcryptCost)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&models.User{}).Where("id = ?", resetToken.UserID).Update("password_hash", pwdHash).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Model(&models.Session{}).Where("user_id = ? AND revoked_at IS NULL", resetToken.UserID).Update("revoked_at", now).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	s.auditor.Record("auth.password_reset.confirm", &resetToken.UserID, map[string]any{})
	return nil
}

func (s *AuthService) isLocked(email, ip string) bool {
	var attempt models.FailedLoginAttempt
	if err := s.db.Where("email = ? AND ip = ?", email, ip).First(&attempt).Error; err != nil {
		return false
	}
	return attempt.LockedUntil.After(time.Now().UTC())
}

func (s *AuthService) recordFailedAttempt(email, ip string) {
	var attempt models.FailedLoginAttempt
	now := time.Now().UTC()
	if err := s.db.Where("email = ? AND ip = ?", email, ip).First(&attempt).Error; err != nil {
		_ = s.db.Create(&models.FailedLoginAttempt{
			Email:       email,
			IP:          ip,
			Failures:    1,
			LockedUntil: now,
		}).Error
		return
	}

	attempt.Failures++
	lockDuration := backoffDuration(attempt.Failures)
	if attempt.Failures >= 5 {
		attempt.LockedUntil = now.Add(lockDuration)
	} else {
		attempt.LockedUntil = now
	}
	attempt.UpdatedAt = now
	_ = s.db.Save(&attempt).Error
}

func (s *AuthService) clearFailedAttempt(email, ip string) {
	_ = s.db.Where("email = ? AND ip = ?", email, ip).Delete(&models.FailedLoginAttempt{}).Error
}

func backoffDuration(failures int) time.Duration {
	if failures < 5 {
		return 0
	}
	seconds := 1 << min(failures-5, 20)
	if seconds > 900 {
		seconds = 900
	}
	return time.Duration(seconds) * time.Second
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
