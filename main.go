type Ping struct {
    ID      uint   `json:"id" gorm:"primaryKey"`
    Message string `json:"message" binding:"required"` // Add validation using binding tag
}

func main() {
    // Initialize the database
    db, err := gorm.Open(sqlite.Open("db/data.db"), &gorm.Config{})
    if err != nil {
        return err // Handle error more gracefully (e.g., return to caller)
    }

    // Auto-migrate the schema
    err = db.AutoMigrate(&Ping{})
    if err != nil {
        return err // Handle error more gracefully
    }

    r := gin.Default()

    // Register routes
    r.POST("/pings", func(c *gin.Context) {
        var ping Ping
        if err := c.ShouldBindJSON(&ping); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ping data"})
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

    // Start the server
    if err := r.Run(":8080"); err != nil {
        return err // Handle error more gracefully
    }
}