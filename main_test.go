package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect test database")
	}
	err = db.AutoMigrate(&Ping{})
	if err != nil {
		panic("failed to migrate test database")
	}
	return db
}

func setupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()
	r.POST("/ping", func(c *gin.Context) {
		var ping Ping
		if err := c.ShouldBindJSON(&ping); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if result := db.Create(&ping); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save ping"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Ping registered successfully!"})
	})
	return r
}

func TestCreatePingSuccess(t *testing.T) {
	db := setupTestDB()
	router := setupRouter(db)

	pingData := Ping{Message: "test message"}
	jsonData, _ := json.Marshal(pingData)
	req, _ := http.NewRequest("POST", "/ping", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["message"] != "Ping registered successfully!" {
		t.Errorf("Expected success message, got %s", response["message"])
	}

	var savedPing Ping
	db.First(&savedPing)
	if savedPing.Message != "test message" {
		t.Errorf("Expected saved message 'test message', got '%s'", savedPing.Message)
	}
}

func TestCreatePingInvalidInput(t *testing.T) {
	db := setupTestDB()
	router := setupRouter(db)

	invalidJSON := `{"message": }`
	req, _ := http.NewRequest("POST", "/ping", bytes.NewBufferString(invalidJSON))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["error"] == "" {
		t.Errorf("Expected error message, got none")
	}

	var count int64
	db.Model(&Ping{}).Count(&count)
	if count != 0 {
		t.Errorf("Expected no data saved, but found %d", count)
	}
}