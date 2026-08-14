package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"tracking-backend/internal/config"
	"tracking-backend/internal/handler"
	"tracking-backend/internal/middleware"
	ws "tracking-backend/internal/websocket"
)

func New(
	cfg *config.Config,
	authHandler *handler.AuthHandler,
	userHandler *handler.UserHandler,
	trackingHandler *handler.TrackingHandler,
	wsHandler *ws.WSHandler,
	rateLimiter *middleware.RateLimiter,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.Logger)
	// CORS: allow all origins/methods/headers
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"*"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Health
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// WebSocket (auth via query param, not header middleware)
	r.Get("/ws/tracking", wsHandler.Handle)

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Auth routes with rate limiting
		r.Group(func(r chi.Router) {
			r.Use(rateLimiter.Middleware)
			r.Post("/auth/register", authHandler.Register)
			r.Post("/auth/login", authHandler.Login)
			r.Post("/auth/refresh", authHandler.Refresh)
		})
		r.Post("/auth/logout", authHandler.Logout)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(cfg))

			r.Get("/users/me", userHandler.Me)
			r.Put("/users/me", userHandler.UpdateMe)

			r.Get("/tracking/history", trackingHandler.History)
			r.Get("/tracking/detail", trackingHandler.Detail)
			r.Post("/tracking/points/batch", trackingHandler.BatchPoints)
			r.Post("/tracking/session/start", trackingHandler.StartSession)
			r.Post("/tracking/session/end", trackingHandler.EndSession)
		})
	})

	return r
}
