package middleware

import (
	"backend/pkg/websocket"
	"database/sql"
	"log"
	"net/http"
	"time"
)

type AppCore struct {
	Hub *websocket.Hub
}

func NewAppCore(db *sql.DB) *AppCore { // Accept DB as a parameter
	return &AppCore{
		Hub: websocket.NewHub(),
	}
}

func (c *AppCore) Close() {
//	c.DB.Close()
	// Add any other cleanup logic here
}

func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Logging
		start := time.Now()
		log.Printf("Started %s %s", r.Method, r.URL.Path)

		// Error handling
		defer func() {
			if err := recover(); err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				log.Printf("Recovered from panic: %v", err)
			}
			log.Printf("Completed %s %s in %v", r.Method, r.URL.Path, time.Since(start))
		}()

		next.ServeHTTP(w, r)
	})
}
