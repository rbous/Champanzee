package main

import (
	"2026champs/internal/cache"
	"2026champs/internal/config"
	"2026champs/internal/repository"
	"2026champs/internal/service"
	"2026champs/internal/transport/rest"
	"2026champs/internal/transport/ws"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// @title 2026 Champs Survey API
// @version 2.0
// @description Kahoot-like async survey room system
// @host localhost:8080
// @BasePath /v1
func main() {
	log.Println("started")
	ctx := context.Background()

	// Load AI config and log model settings
	aiConfig := config.DefaultAIConfig()
	log.Printf("AI Config:")
	log.Printf("  L1 Eval:   %s", aiConfig.Models.L1Eval)
	log.Printf("  FollowUp:  %s", aiConfig.Models.FollowUp)
	log.Printf("  L3 Refresh: %s", aiConfig.Models.L3Refresh)
	log.Printf("  Pool Gen:  %s", aiConfig.Models.PoolGen)
	log.Printf("  Report:    %s", aiConfig.Models.Report)
	if aiConfig.IsEnabled() {
		log.Println("  API Key:   configured ✓")
	} else {
		log.Println("  API Key:   NOT SET (using mock evaluator)")
	}

	// MongoDB connection
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://admin:password@mongodb:27017/champsdb?authSource=admin"
		log.Println("Warning: MONGO_URI not set, using default")
	}

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer mongoClient.Disconnect(ctx)

	// Ping MongoDB
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := mongoClient.Ping(pingCtx, nil); err != nil {
		log.Fatal("Failed to ping MongoDB:", err)
	}
	log.Println("Connected to MongoDB")

	db := mongoClient.Database("champsdb")

	// Redis connection
	redisAddr := os.Getenv("REDIS_URI")
	if redisAddr == "" {
		redisAddr = "redis:6379"
		log.Println("Warning: REDIS_URI not set, using default")
	}
	// Remove redis:// prefix if present
	if len(redisAddr) > 8 && redisAddr[:8] == "redis://" {
		redisAddr = redisAddr[8:]
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	defer rdb.Close()

	// Ping Redis
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		log.Fatal("Failed to ping Redis:", err)
	}
	log.Println("Connected to Redis")

	// Initialize WebSocket hub
	wsHub := ws.NewHub()
	log.Println("WebSocket hub started")

	// Initialize repositories
	surveyRepo := repository.NewSurveyRepo(db)
	roomRepo := repository.NewRoomRepo(db)
	answerRepo := repository.NewAnswerRepo(db)
	reportRepo := repository.NewReportRepo(db)

	// Initialize caches
	roomCache := cache.NewRoomCache(rdb)
	playerCache := cache.NewPlayerCache(rdb)
	leaderboard := cache.NewLeaderboardCache(rdb)
	poolCache := cache.NewPoolCache(rdb)
	analyticsCache := cache.NewAnalyticsCache(rdb)

	// Initialize services
	authSvc := service.NewAuthService()
	surveySvc := service.NewSurveyService(surveyRepo)
	evaluator := service.NewEvaluatorService()
	reportSvc := service.NewReportService(roomRepo, answerRepo, reportRepo, analyticsCache, leaderboard, evaluator)
	roomSvc := service.NewRoomService(roomRepo, surveyRepo, roomCache, authSvc, reportSvc)
	playerSvc := service.NewPlayerService(surveyRepo, roomCache, playerCache, leaderboard, authSvc)
	analyticsSvc := service.NewAnalyticsService(analyticsCache, evaluator)
	answerSvc := service.NewAnswerService(answerRepo, playerCache, poolCache, playerSvc, evaluator)

	// Inject analytics service into answer service for L2/L3/L4 updates
	answerSvc.SetAnalyticsService(analyticsSvc)

	// Inject broadcaster (wsHub implements service.Broadcaster)
	answerSvc.SetBroadcaster(wsHub)
	playerSvc.SetBroadcaster(wsHub)
	roomSvc.SetBroadcaster(wsHub)

	// Create router with container
	container := &rest.Container{
		AuthService:   authSvc,
		SurveyService: surveySvc,
		RoomService:   roomSvc,
		PlayerService: playerSvc,
		AnswerService: answerSvc,
		ReportService: reportSvc,
		Leaderboard:   leaderboard,
		WSHub:         wsHub,
	}

	router := rest.NewRouter(container)

	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start server
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		log.Printf("Server starting on :%s", port)
		log.Printf("Host auth: username=%s", os.Getenv("HOST_USERNAME"))
		log.Println("Endpoints:")
		log.Println("  POST /v1/auth/login")
		log.Println("  POST/GET /v1/surveys")
		log.Println("  POST/GET /v1/rooms")
		log.Println("  POST /v1/rooms/{code}/join")
		log.Println("  GET  /v1/reports/{code}/snapshot")
		log.Println("  GET/POST /v1/reports/{code}/ai")
		log.Println("  WS  /v1/ws/rooms/{code}/host")
		log.Println("  WS  /v1/ws/rooms/{code}/player")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("ListenAndServe:", err)
		}
	}()

	// Wait for interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
