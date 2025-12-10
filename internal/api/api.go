package api

import (
	"errors"
	"strings"

	"github.com/nk2580/Tester-API/internal/store"
)

// ErrValidation is returned when input validation fails
var ErrValidation = errors.New("validation error")

// ErrEmptyMessage is returned when a ping message is empty
var ErrEmptyMessage = errors.New("validation: message required")

// CreatePing validates and saves a ping to the store
// Returns ErrEmptyMessage if the message is empty after trimming
// Returns storage errors if the save operation fails
func CreatePing(s store.Store, p *store.Ping) error {
	// Validate: message must be non-empty after trimming
	p.Message = strings.TrimSpace(p.Message)
	if p.Message == "" {
		return ErrEmptyMessage
	}

	// Save to store
	if err := s.SavePing(p); err != nil {
		return err
	}

	return nil
}

// ListPings retrieves all pings from the store
func ListPings(s store.Store) ([]store.Ping, error) {
	pings, err := s.GetPings()
	if err != nil {
		return nil, err
	}
	
	return pings, nil
}
