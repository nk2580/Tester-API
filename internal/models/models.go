package models

import "time"

type Ping struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type User struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	Email        string     `json:"email" gorm:"uniqueIndex;size:320;not null"`
	PasswordHash string     `json:"-" gorm:"not null"`
	IsVerified   bool       `json:"is_verified" gorm:"not null;default:true"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type PasswordResetToken struct {
	ID        uint       `gorm:"primaryKey"`
	UserID    uint       `gorm:"not null;index"`
	User      User       `gorm:"constraint:OnDelete:CASCADE"`
	TokenHash string     `gorm:"not null;size:64;index"`
	ExpiresAt time.Time  `gorm:"not null;index"`
	UsedAt    *time.Time `gorm:"index"`
	CreatedAt time.Time
}

type Session struct {
	ID        uint       `gorm:"primaryKey"`
	UserID    uint       `gorm:"not null;index"`
	User      User       `gorm:"constraint:OnDelete:CASCADE"`
	TokenHash string     `gorm:"not null;size:64;uniqueIndex"`
	ExpiresAt time.Time  `gorm:"not null;index"`
	RevokedAt *time.Time `gorm:"index"`
	CreatedAt time.Time
}

type FailedLoginAttempt struct {
	ID          uint      `gorm:"primaryKey"`
	Email       string    `gorm:"not null;index:idx_failed_login_scope,priority:1;size:320"`
	IP          string    `gorm:"not null;index:idx_failed_login_scope,priority:2;size:64"`
	Failures    int       `gorm:"not null;default:0"`
	LockedUntil time.Time `gorm:"not null"`
	UpdatedAt   time.Time
	CreatedAt   time.Time
}

type AuditEvent struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    *uint     `gorm:"index"`
	EventType string    `gorm:"not null;index;size:64"`
	Metadata  string    `gorm:"not null;type:text"`
	CreatedAt time.Time `gorm:"index"`
}
