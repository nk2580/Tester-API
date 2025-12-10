package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates a test database in memory for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	err = db.AutoMigrate(&Ping{})
	if err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}

// setupRouter creates a test router with the given database
func setupRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	// Register routes (same as main)
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

func TestPing_Model(t *testing.T) {
	ping := Ping{
		ID:      1,
		Message: "test message",
	}

	assert.Equal(t, uint(1), ping.ID)
	assert.Equal(t, "test message", ping.Message)
}

func TestPOST_Ping_Success(t *testing.T) {
	db := setupTestDB(t)
	router := setupRouter(db)

	// Create a test ping
	ping := Ping{
		Message: "Hello, World!",
	}
	jsonData, _ := json.Marshal(ping)

	// Create request
	req, _ := http.NewRequest("POST", "/ping", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Record response
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Ping registered successfully!", response["message"])

	// Verify the ping was saved to the database
	var savedPing Ping
	result := db.First(&savedPing)
	assert.NoError(t, result.Error)
	assert.Equal(t, "Hello, World!", savedPing.Message)
}

func TestPOST_Ping_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	router := setupRouter(db)

	// Create invalid JSON
	invalidJSON := []byte(`{"message": }`)

	// Create request
	req, _ := http.NewRequest("POST", "/ping", bytes.NewBuffer(invalidJSON))
	req.Header.Set("Content-Type", "application/json")

	// Record response
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "invalid")
}

func TestPOST_Ping_EmptyMessage(t *testing.T) {
	db := setupTestDB(t)
	router := setupRouter(db)

	// Create a ping with empty message
	ping := Ping{
		Message: "",
	}
	jsonData, _ := json.Marshal(ping)

	// Create request
	req, _ := http.NewRequest("POST", "/ping", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Record response
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert response - empty message should still be accepted
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Ping registered successfully!", response["message"])
}

func TestGET_Pings_EmptyDatabase(t *testing.T) {
	db := setupTestDB(t)
	router := setupRouter(db)

	// Create request
	req, _ := http.NewRequest("GET", "/pings", nil)

	// Record response
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var pings []Ping
	err := json.Unmarshal(w.Body.Bytes(), &pings)
	assert.NoError(t, err)
	assert.Empty(t, pings)
}

func TestGET_Pings_WithData(t *testing.T) {
	db := setupTestDB(t)
	router := setupRouter(db)

	// Insert test data
	testPings := []Ping{
		{Message: "First ping"},
		{Message: "Second ping"},
		{Message: "Third ping"},
	}
	for _, ping := range testPings {
		db.Create(&ping)
	}

	// Create request
	req, _ := http.NewRequest("GET", "/pings", nil)

	// Record response
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var pings []Ping
	err := json.Unmarshal(w.Body.Bytes(), &pings)
	assert.NoError(t, err)
	assert.Len(t, pings, 3)
	assert.Equal(t, "First ping", pings[0].Message)
	assert.Equal(t, "Second ping", pings[1].Message)
	assert.Equal(t, "Third ping", pings[2].Message)
}

func TestPOST_Then_GET_Integration(t *testing.T) {
	db := setupTestDB(t)
	router := setupRouter(db)

	// First, POST a ping
	ping := Ping{
		Message: "Integration test message",
	}
	jsonData, _ := json.Marshal(ping)

	postReq, _ := http.NewRequest("POST", "/ping", bytes.NewBuffer(jsonData))
	postReq.Header.Set("Content-Type", "application/json")
	postW := httptest.NewRecorder()
	router.ServeHTTP(postW, postReq)

	assert.Equal(t, http.StatusOK, postW.Code)

	// Then, GET all pings
	getReq, _ := http.NewRequest("GET", "/pings", nil)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)

	assert.Equal(t, http.StatusOK, getW.Code)

	var pings []Ping
	err := json.Unmarshal(getW.Body.Bytes(), &pings)
	assert.NoError(t, err)
	assert.Len(t, pings, 1)
	assert.Equal(t, "Integration test message", pings[0].Message)
}

func TestPOST_Multiple_Pings(t *testing.T) {
	db := setupTestDB(t)
	router := setupRouter(db)

	// POST multiple pings
	messages := []string{"First", "Second", "Third", "Fourth"}
	for _, msg := range messages {
		ping := Ping{Message: msg}
		jsonData, _ := json.Marshal(ping)

		req, _ := http.NewRequest("POST", "/ping", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Verify all pings were saved
	getReq, _ := http.NewRequest("GET", "/pings", nil)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)

	var pings []Ping
	err := json.Unmarshal(getW.Body.Bytes(), &pings)
	assert.NoError(t, err)
	assert.Len(t, pings, 4)
}
