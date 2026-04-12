package models

import "time"

type UserState string

const (
	UserStatePendingVerification UserState = "pending_verification"
	UserStateActive              UserState = "active"
	UserStateLocked              UserState = "locked"
)

type User struct {
	ID                     uint      `gorm:"primaryKey"`
	Email                  string    `gorm:"uniqueIndex;size:320;not null"`
	PasswordHash           string    `gorm:"size:255;not null"`
	State                  UserState `gorm:"size:32;index;not null"`
	EmailVerifiedAt        *time.Time
	FailedLoginAttempts    int `gorm:"not null;default:0"`
	LockedUntil            *time.Time
	LastVerificationSentAt *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type Session struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"index;not null"`
	TokenHash string    `gorm:"size:128;uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"index;not null"`
	RevokedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type VerificationToken struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"index;not null"`
	TokenHash string    `gorm:"size:128;uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"index;not null"`
	UsedAt    *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type PasswordResetToken struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"index;not null"`
	TokenHash string    `gorm:"size:128;uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"index;not null"`
	UsedAt    *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AuditEvent struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    *uint  `gorm:"index"`
	EventType string `gorm:"size:64;index;not null"`
	IP        string `gorm:"size:64"`
	Details   string `gorm:"type:text"`
	CreatedAt time.Time
}

type Ping struct {
	ID      uint   `json:"id" gorm:"primaryKey"`
	Message string `json:"message"`
}
