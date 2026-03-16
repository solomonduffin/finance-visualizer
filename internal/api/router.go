// Package api provides the chi HTTP router and route registration for the finance-visualizer.
package api

import (
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/go-chi/httplog/v3"
	"github.com/go-chi/httprate"
	"github.com/go-chi/jwtauth/v5"
	"github.com/solomon/finance-visualizer/internal/api/handlers"
)

// NewRouter creates and configures the chi router with:
//   - Request logging middleware (httplog)
//   - CORS middleware (allows localhost:5173 for Vite dev server)
//   - Public login route with IP rate limiting (5 requests per 30 seconds)
//   - Protected route group requiring valid JWT cookie named "jwt"
func NewRouter(tokenAuth *jwtauth.JWTAuth, database *sql.DB) http.Handler {
	r := chi.NewRouter()

	// Request logger middleware
	logger := slog.Default()
	r.Use(httplog.RequestLogger(logger, &httplog.Options{
		Level: slog.LevelInfo,
	}))

	// CORS middleware — allow Vite dev server origin with credentials
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Public routes — rate limited login only
	r.Group(func(r chi.Router) {
		r.With(httprate.LimitByIP(5, 30*time.Second)).
			Post("/api/auth/login", handlers.Login(database))
	})

	// Protected routes — JWT authentication required
	r.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(tokenAuth))
		r.Use(jwtauth.Authenticator(tokenAuth))
		r.Get("/api/health", handlers.Health)
		r.Get("/api/settings", handlers.GetSettings(database))
		r.Post("/api/settings", handlers.SaveSettings(database))
		r.Post("/api/sync/now", handlers.SyncNow(database))
		r.Get("/api/summary", handlers.GetSummary(database))
		r.Get("/api/accounts", handlers.GetAccounts(database))
		r.Patch("/api/accounts/{id}", handlers.UpdateAccount(database))
		r.Get("/api/balance-history", handlers.GetBalanceHistory(database))
		r.Get("/api/sync-log", handlers.GetSyncLog(database))
	})

	return r
}
