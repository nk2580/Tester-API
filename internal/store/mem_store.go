package store

import (
	"errors"
	"sync"
)

// MemoryStore is an in-memory implementation of the Store interface
// for testing purposes. It is thread-safe using a mutex.
type MemoryStore struct {
	mu      sync.Mutex
	pings   []Ping
	nextID  uint
	
	// ErrorOnSave can be set to simulate storage errors during testing
	ErrorOnSave error
}

// NewMemoryStore creates a new in-memory store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		pings:  make([]Ping, 0),
		nextID: 1,
	}
}

// SavePing stores a ping in memory
func (m *MemoryStore) SavePing(ping *Ping) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if error injection is enabled for testing
	if m.ErrorOnSave != nil {
		return m.ErrorOnSave
	}
	
	// Assign an ID if not set
	if ping.ID == 0 {
		ping.ID = m.nextID
		m.nextID++
	}
	
	m.pings = append(m.pings, *ping)
	return nil
}

// GetPings retrieves all pings from memory
func (m *MemoryStore) GetPings() ([]Ping, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Return a copy to prevent external modification
	result := make([]Ping, len(m.pings))
	copy(result, m.pings)
	
	return result, nil
}

// Clear removes all pings from the store (useful for test cleanup)
func (m *MemoryStore) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.pings = make([]Ping, 0)
	m.nextID = 1
	m.ErrorOnSave = nil
}

// Seed adds pings to the store (useful for test setup)
func (m *MemoryStore) Seed(pings []Ping) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for _, ping := range pings {
		if ping.ID == 0 {
			ping.ID = m.nextID
			m.nextID++
		} else if ping.ID >= m.nextID {
			m.nextID = ping.ID + 1
		}
		m.pings = append(m.pings, ping)
	}
}

// SetErrorOnSave configures the store to return an error on the next Save call
func (m *MemoryStore) SetErrorOnSave(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ErrorOnSave = err
}

// Common error for testing storage failures
var ErrStorageFailure = errors.New("storage operation failed")
