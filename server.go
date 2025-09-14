package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()

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

	return r
}
