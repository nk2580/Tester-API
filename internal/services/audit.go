package services

import (
	"encoding/json"

	"github.com/nk2580/Tester-API/internal/models"
	"gorm.io/gorm"
)

type AuditService struct {
	db *gorm.DB
}

func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
}

func (s *AuditService) Record(eventType string, userID *uint, metadata map[string]any) {
	blob, _ := json.Marshal(metadata)
	_ = s.db.Create(&models.AuditEvent{EventType: eventType, UserID: userID, Metadata: string(blob)}).Error
}
