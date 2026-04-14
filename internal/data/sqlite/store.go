package sqlite

import (
    "context"

    "github.com/nk2580/Tester-API/internal/data"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

// Open opens a SQLite database located at path and configures gorm defaults.
func Open(path string) (*gorm.DB, error) {
    return gorm.Open(sqlite.Open(path), &gorm.Config{TranslateError: true})
}

// Store implements the data.UserStore and data.PingStore interfaces on top of gorm.
type Store struct {
    db *gorm.DB
}

// NewStore wraps the provided gorm DB handle.
func NewStore(db *gorm.DB) *Store {
    return &Store{db: db}
}

// AutoMigrate applies the schema for the core models.
func (s *Store) AutoMigrate() error {
    return s.db.AutoMigrate(&data.Ping{}, &data.User{})
}

// CreatePing persists a ping record.
func (s *Store) CreatePing(ctx context.Context, ping *data.Ping) error {
    return s.db.WithContext(ctx).Create(ping).Error
}

// ListPings fetches all ping rows.
func (s *Store) ListPings(ctx context.Context) ([]data.Ping, error) {
    var pings []data.Ping
    if err := s.db.WithContext(ctx).Find(&pings).Error; err != nil {
        return nil, err
    }
    return pings, nil
}

// CreateUser persists a user row.
func (s *Store) CreateUser(ctx context.Context, user *data.User) error {
    return s.db.WithContext(ctx).Create(user).Error
}

// FindByEmail looks up a user by email.
func (s *Store) FindByEmail(ctx context.Context, email string) (*data.User, error) {
    var user data.User
    if err := s.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
        return nil, err
    }
    return &user, nil
}

// FindByID looks up a user by primary key.
func (s *Store) FindByID(ctx context.Context, id uint) (*data.User, error) {
    var user data.User
    if err := s.db.WithContext(ctx).First(&user, id).Error; err != nil {
        return nil, err
    }
    return &user, nil
}

// DB exposes the underlying gorm DB handle for tests.
func (s *Store) DB() *gorm.DB {
    return s.db
}
