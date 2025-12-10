package store

// Store defines the interface for ping persistence operations
type Store interface {
	// SavePing persists a ping record
	SavePing(ping *Ping) error
	
	// GetPings retrieves all ping records
	GetPings() ([]Ping, error)
}
