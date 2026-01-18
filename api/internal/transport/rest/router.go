package rest

import (
	"2026champs/internal/cache"
	"2026champs/internal/service"
	"2026champs/internal/transport/rest/handler"
	"2026champs/internal/transport/rest/middleware"
	"2026champs/internal/transport/ws"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

// Container holds all dependencies for the router
type Container struct {
	AuthService   *service.AuthService
	SurveyService *service.SurveyService
	RoomService   *service.RoomService
	PlayerService *service.PlayerService
	AnswerService *service.AnswerService
	ReportService *service.ReportService
	Leaderboard   cache.LeaderboardCache
	WSHub         *ws.Hub
}

// NewRouter creates the API router with all endpoints
func NewRouter(c *Container) http.Handler {
	r := mux.NewRouter()

	// Initialize handlers
	authHandler := handler.NewAuthHandler(c.AuthService)
	surveyHandler := handler.NewSurveyHandler(c.SurveyService)
	roomHandler := handler.NewRoomHandler(c.RoomService, c.PlayerService, c.Leaderboard)
	playerHandler := handler.NewPlayerHandler(c.PlayerService, c.AnswerService)
	reportHandler := handler.NewReportHandler(c.ReportService)
	wsHandler := ws.NewHandler(c.WSHub, c.AuthService, c.PlayerService)

	// Initialize middleware
	authMW := middleware.NewAuthMiddleware(c.AuthService)

	// CORS middleware (apply first)
	r.Use(corsMiddleware)

	// API v1 routes
	v1 := r.PathPrefix("/v1").Subrouter()

	// Public routes
	v1.HandleFunc("/auth/login", authHandler.Login).Methods("POST", "OPTIONS")
	v1.HandleFunc("/rooms/{code}/join", roomHandler.Join).Methods("POST", "OPTIONS")

	// WebSocket routes (public with token in query param)
	v1.HandleFunc("/ws/rooms/{code}/host", wsHandler.HostWS).Methods("GET")
	v1.HandleFunc("/ws/rooms/{code}/player", wsHandler.PlayerWS).Methods("GET")

	// Health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}).Methods("GET")

	// Host routes (require host auth)
	hostRoutes := v1.NewRoute().Subrouter()
	hostRoutes.Use(authMW.RequireHost)

	hostRoutes.HandleFunc("/surveys", surveyHandler.Create).Methods("POST", "OPTIONS")
	hostRoutes.HandleFunc("/surveys", surveyHandler.List).Methods("GET", "OPTIONS")
	hostRoutes.HandleFunc("/surveys/{surveyId}", surveyHandler.Get).Methods("GET", "OPTIONS")
	hostRoutes.HandleFunc("/surveys/{surveyId}", surveyHandler.Update).Methods("PUT", "OPTIONS")
	hostRoutes.HandleFunc("/rooms", roomHandler.Create).Methods("POST", "OPTIONS")
	hostRoutes.HandleFunc("/rooms/{code}", roomHandler.Get).Methods("GET", "OPTIONS")
	hostRoutes.HandleFunc("/rooms/{code}/start", roomHandler.Start).Methods("POST", "OPTIONS")
	hostRoutes.HandleFunc("/rooms/{code}/end", roomHandler.End).Methods("POST", "OPTIONS")
	hostRoutes.HandleFunc("/rooms/{code}/leaderboard", roomHandler.Leaderboard).Methods("GET", "OPTIONS")

	// Report routes (host only)
	hostRoutes.HandleFunc("/reports/{roomCode}/snapshot", reportHandler.GetSnapshot).Methods("GET", "OPTIONS")
	hostRoutes.HandleFunc("/reports/{roomCode}/ai", reportHandler.GetAIReport).Methods("GET", "OPTIONS")
	hostRoutes.HandleFunc("/reports/{roomCode}/ai", reportHandler.GenerateAIReport).Methods("POST", "OPTIONS")

	// Player routes (require player auth)
	playerRoutes := v1.NewRoute().Subrouter()
	playerRoutes.Use(authMW.RequirePlayer)

	playerRoutes.HandleFunc("/rooms/{code}/question/current", playerHandler.GetCurrentQuestion).Methods("GET", "OPTIONS")
	playerRoutes.HandleFunc("/rooms/{code}/questions/{questionKey}/draft", playerHandler.SaveDraft).Methods("PUT", "OPTIONS")
	playerRoutes.HandleFunc("/rooms/{code}/answers", playerHandler.SubmitAnswer).Methods("POST", "OPTIONS")
	playerRoutes.HandleFunc("/rooms/{code}/questions/{questionKey}/skip", playerHandler.Skip).Methods("POST", "OPTIONS")

	return r
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
		if allowedOrigins == "" {
			allowedOrigins = "*"
		}

		allowedMethods := os.Getenv("CORS_ALLOWED_METHODS")
		if allowedMethods == "" {
			allowedMethods = "GET, POST, PUT, DELETE, OPTIONS"
		}

		allowedHeaders := os.Getenv("CORS_ALLOWED_HEADERS")
		if allowedHeaders == "" {
			allowedHeaders = "Content-Type, Authorization"
		}

		w.Header().Set("Access-Control-Allow-Origin", allowedOrigins)
		w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
		w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
