package store

import (
	"fmt"

	"github.com/nk2580/Tester-API/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func OpenSQLite(path string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.AutoMigrate(
		&models.Ping{},
		&models.User{},
		&models.Session{},
		&models.VerificationToken{},
		&models.PasswordResetToken{},
		&models.AuditEvent{},
	); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}
	return db, nil
}
