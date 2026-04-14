package data

import "context"

// UserStore abstracts user persistence operations.
type UserStore interface {
	CreateUser(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id uint) (*User, error)
}

// PingStore abstracts ping persistence operations.
type PingStore interface {
	CreatePing(ctx context.Context, ping *Ping) error
	ListPings(ctx context.Context) ([]Ping, error)
}
