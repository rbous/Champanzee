package main

import (
	"2026champs/config"
	"2026champs/internal/app"
	"2026champs/internal/cache"
	"2026champs/internal/repository"
	"2026champs/internal/transport/rest"
	mongostore "2026champs/store/mongo"
	redisstore "2026champs/store/redis"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()

	// Initialize Mongo
	mongoClient, err := mongostore.NewMongoClient(cfg.MongoURI)
	if err != nil {
		log.Fatal("Failed to connect to Mongo:", err)
	}
	defer mongoClient.Disconnect(ctx)

	// Initialize Redis
	redisClient := redisstore.NewRedisClient(cfg.RedisAddr)

	// Initialize repos
	playerRepo := repository.NewPlayerRepo(mongoClient)
	roomRepo := repository.NewRoomRepo(mongoClient)
	sessionRepo := repository.NewSessionRepo(mongoClient)
	sessionCache := cache.NewSessionCache(redisClient)

	// Initialize app
	app := &app.App{
		PlayerRepo:   playerRepo,
		RoomRepo:     roomRepo,
		SessionRepo:  sessionRepo,
		SessionCache: sessionCache,
	}

	// Initialize HTTP router
	router := rest.NewRouter(app)

	// Start server
	srv := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: router,
	}

	go func() {
		log.Printf("Starting HTTP server on port %s", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("ListenAndServe:", err)
		}
	}()

	// Wait for interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
