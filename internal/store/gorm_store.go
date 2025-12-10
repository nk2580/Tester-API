package store

import "gorm.io/gorm"

// GormStore is a production implementation of the Store interface
// that wraps a GORM database connection
type GormStore struct {
	db *gorm.DB
}

// NewGormStore creates a new GORM-backed store
func NewGormStore(db *gorm.DB) *GormStore {
	return &GormStore{db: db}
}

// SavePing persists a ping to the database
func (g *GormStore) SavePing(ping *Ping) error {
	result := g.db.Create(ping)
	return result.Error
}

// GetPings retrieves all pings from the database
func (g *GormStore) GetPings() ([]Ping, error) {
	var pings []Ping
	result := g.db.Find(&pings)
	if result.Error != nil {
		return nil, result.Error
	}
	return pings, nil
}
