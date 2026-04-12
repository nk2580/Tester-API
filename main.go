package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Ping struct {
	ID      uint   `json:"id" gorm:"primaryKey"`
	Message string `json:"message"`
}

var db *gorm.DB

func main() {
	var err error
	db, err = gorm.Open(sqlite.Open("db/data.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(&Ping{})
	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	http.HandleFunc("/ping", handlePing)
	http.HandleFunc("/pings", handlePings)
	http.HandleFunc("/hello", handleHello)
	http.HandleFunc("/time", handleTime)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
		return
	}

	var ping Ping
	if err := json.NewDecoder(r.Body).Decode(&ping); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
		return
	}

	if result := db.Create(&ping); result.Error != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "Failed to save ping"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"message": "Ping registered successfully!"})
}

func handlePings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
		return
	}

	var pings []Ping
	if result := db.Find(&pings); result.Error != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "Failed to retrieve pings"})
		return
	}

	respondJSON(w, http.StatusOK, pings)
}

func handleHello(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"message": "Hello World"})
}

func handleTime(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
		return
	}

	timezone := r.Header.Get("X-Timezone")
	if timezone == "" {
		timezone = "UTC"
	}

	location, err := time.LoadLocation(timezone)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "Invalid timezone"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"time":     time.Now().In(location).Format(time.RFC3339),
		"timezone": timezone,
	})
}
