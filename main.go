package main

import (
	"encoding/json"
	"errors"
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

	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var ping Ping
		if err := decodeJSONBody(r, &ping); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		// Save the ping to the database
		if result := db.Create(&ping); result.Error != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save ping"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"message": "Ping registered successfully!"})
	})

	mux.HandleFunc("/pings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var pings []Ping

		// Retrieve all pings from the database
		if result := db.Find(&pings); result.Error != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve pings"})
			return
		}

		writeJSON(w, http.StatusOK, pings)
	})

	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"message": "Hello World"})
	})

	mux.HandleFunc("/time", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

	// Start the server
	if err := http.ListenAndServe(":8080", withRecovery(mux)); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}

func decodeJSONBody(r *http.Request, v any) error {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(v); err != nil {
		return err
	}

	if decoder.More() {
		return errors.New("invalid JSON body")
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to encode JSON response: %v", err)
	}
}

func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
