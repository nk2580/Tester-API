package store

// Ping represents a ping record with a message
type Ping struct {
	ID      uint   `json:"id" gorm:"primaryKey"`
	Message string `json:"message"`
}
