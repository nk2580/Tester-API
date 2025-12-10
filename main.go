package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nk2580/Tester-API/internal/api"
	"github.com/nk2580/Tester-API/internal/store"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// Initialize the database
	db, err := gorm.Open(sqlite.Open("db/data.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(&store.Ping{})
	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	// Create the store implementation
	pingStore := store.NewGormStore(db)

	r := gin.Default()

	// Register routes
	r.POST("/ping", func(c *gin.Context) {
		var ping store.Ping
		if err := c.ShouldBindJSON(&ping); err != nil {
			log.Printf("failed to parse JSON: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Use business logic API to create ping
		if err := api.CreatePing(pingStore, &ping); err != nil {
			// Map validation errors to 400 Bad Request
			if errors.Is(err, api.ErrEmptyMessage) {
				log.Printf("validation failed: %v", err)
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			
			// Map storage errors to 500 Internal Server Error
			log.Printf("save ping failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save ping"})
			return
		}

		log.Printf("save ping succeeded: id=%d, message=%s", ping.ID, ping.Message)
		c.JSON(http.StatusOK, gin.H{"message": "Ping registered successfully!"})
	})

	r.GET("/pings", func(c *gin.Context) {
		// Use business logic API to list pings
		pings, err := api.ListPings(pingStore)
		if err != nil {
			log.Printf("list pings failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve pings"})
			return
		}

		log.Printf("list pings succeeded: count=%d", len(pings))
		c.JSON(http.StatusOK, pings)
	})

	// Start the server
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
