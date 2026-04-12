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

func main() {
	db, err := gorm.Open(sqlite.Open("db/data.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(&Ping{})
	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		var ping Ping
		if err := json.NewDecoder(r.Body).Decode(&ping); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if result := db.Create(&ping); result.Error != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save ping"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"message": "Ping registered successfully!"})
	})

	mux.HandleFunc("/pings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		var pings []Ping

		if result := db.Find(&pings); result.Error != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve pings"})
			return
		}

		writeJSON(w, http.StatusOK, pings)
	})

	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"message": "Hello World"})
	})

	mux.HandleFunc("/time", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		timezone := r.Header.Get("X-Timezone")
		if timezone == "" {
			timezone = "UTC"
		}

		location, err := time.LoadLocation(timezone)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid timezone"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"time":     time.Now().In(location).Format(time.RFC3339),
			"timezone": timezone,
		})
	})

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}
