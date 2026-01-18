package rest

import (
	"2026champs/internal/app"
	"2026champs/internal/transport/rest/handlers"
	"net/http"

	"github.com/gorilla/mux"
)

func NewRouter(app *app.App) http.Handler {
	r := mux.NewRouter()

	// API prefix
	api := r.PathPrefix("/api").Subrouter()

	// Room endpoints
	api.HandleFunc("/rooms", handlers.CreateRoom(app)).Methods("POST")
	api.HandleFunc("/rooms/{code}", handlers.GetRoom(app)).Methods("GET")
	api.HandleFunc("/rooms/join", handlers.JoinRoom(app)).Methods("POST")

	// Session endpoints
	api.HandleFunc("/sessions", handlers.StartSession(app)).Methods("POST")
	api.HandleFunc("/sessions/{id}", handlers.GetSession(app)).Methods("GET")
	api.HandleFunc("/sessions/{id}/end", handlers.EndSession(app)).Methods("POST")

	// Answer endpoints
	api.HandleFunc("/answers", handlers.SubmitAnswer(app)).Methods("POST")
	api.HandleFunc("/sessions/{sessionId}/answers", handlers.GetAnswersBySession(app)).Methods("GET")

	// Player endpoints
	api.HandleFunc("/players", handlers.GetPlayers(app)).Methods("GET")

	// CORS middleware for development
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	})

	return r
}
