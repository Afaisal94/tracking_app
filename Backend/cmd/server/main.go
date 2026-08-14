package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"tracking-backend/internal/config"
	"tracking-backend/internal/database"
	"tracking-backend/internal/handler"
	"tracking-backend/internal/middleware"
	"tracking-backend/internal/repository"
	"tracking-backend/internal/router"
	"tracking-backend/internal/service"
	ws "tracking-backend/internal/websocket"
	"tracking-backend/internal/worker"
)

func main() {
	// Logger setup
	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	cfg := config.Load()
	log.Info().Str("env", cfg.Env).Str("addr", cfg.Addr()).Msg("starting server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Database
	pool, err := database.NewPostgresPool(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to postgres")
	}
	defer pool.Close()
	log.Info().Msg("connected to postgres")

	// Migrations
	if err := database.RunMigrations(ctx, pool, "migrations"); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}

	// Redis
	redisClient, err := database.NewRedisClient(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis")
	}
	defer redisClient.Close()
	log.Info().Msg("connected to redis")

	// Repositories
	userRepo := repository.NewUserRepo(pool)
	tokenRepo := repository.NewTokenRepo(pool)
	trackingRepo := repository.NewTrackingRepo(pool)

	// Services
	redisSvc := service.NewRedisService(redisClient)
	authSvc := service.NewAuthService(cfg, userRepo, tokenRepo)
	trackingSvc := service.NewTrackingService(trackingRepo)

	// Handlers
	authHandler := handler.NewAuthHandler(authSvc, trackingSvc, redisSvc)
	userHandler := handler.NewUserHandler(userRepo)
	trackingHandler := handler.NewTrackingHandler(trackingSvc)

	// WebSocket
	wsHandler := ws.NewWSHandler(cfg, trackingRepo, redisSvc)

	// Rate limiter
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitAuthRPS, cfg.RateLimitAuthBurst)
	rateLimiter.StartCleanup(5 * time.Minute)

	// Router
	h := router.New(cfg, authHandler, userHandler, trackingHandler, wsHandler, rateLimiter)

	server := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      h,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Background workers
	bgCtx, bgCancel := context.WithCancel(context.Background())
	defer bgCancel()

	go wsHandler.StartFlushWorker(bgCtx)
	go wsHandler.ListenForceDisconnect(bgCtx)

	cleanupWorker := worker.NewCleanupWorker(redisSvc, trackingRepo, cfg.WSIdleTimeout)
	go cleanupWorker.Start(bgCtx)

	// Start server
	go func() {
		log.Info().Str("addr", cfg.Addr()).Msg("server listening")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down server...")

	bgCancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("server forced to shutdown")
	}
	log.Info().Msg("server exited")
}
