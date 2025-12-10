package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
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
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Save the ping using the store
		if err := pingStore.SavePing(&ping); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save ping"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Ping registered successfully!"})
	})

	r.GET("/pings", func(c *gin.Context) {
		// Retrieve all pings using the store
		pings, err := pingStore.GetPings()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve pings"})
			return
		}

		c.JSON(http.StatusOK, pings)
	})

	// Start the server
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
