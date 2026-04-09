package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Ping struct {
	ID      uint   `json:"id" gorm:"primaryKey"`
	Message string `json:"message"`
}

func main() {
	// Initialize the database
	db, err := gorm.Open(sqlite.Open("db/data.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(&Ping{})
	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	r := gin.Default()

	// Register routes
	r.POST("/ping", func(c *gin.Context) {
		var ping Ping
		if err := c.ShouldBindJSON(&ping); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Save the ping to the database
		if result := db.Create(&ping); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save ping"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Ping registered successfully!"})
	})

	r.GET("/pings", func(c *gin.Context) {
		var pings []Ping

		// Retrieve all pings from the database
		if result := db.Find(&pings); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve pings"})
			return
		}

		c.JSON(http.StatusOK, pings)
	})

	r.GET("/hello", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Hello World"})
	})

	r.GET("/time", func(c *gin.Context) {
		timezone := c.GetHeader("X-Timezone")
		if timezone == "" {
			timezone = "UTC"
		}

		location, err := time.LoadLocation(timezone)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timezone. Use a valid IANA timezone (for example, America/New_York)."})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"timezone": timezone,
			"time":     time.Now().In(location).Format(time.RFC3339),
		})
	})

	// Start the server
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
