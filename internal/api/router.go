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
func NewRouter(tokenAuth *jwtauth.JWTAuth, database *sql.DB, jwtSecret string) http.Handler {
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
		r.Post("/api/settings", handlers.SaveSettings(database, jwtSecret))
		r.Post("/api/sync/now", handlers.SyncNow(database, jwtSecret))
		r.Get("/api/summary", handlers.GetSummary(database))
		r.Get("/api/accounts", handlers.GetAccounts(database))
		r.Patch("/api/accounts/{id}", handlers.UpdateAccount(database))
		r.Get("/api/balance-history", handlers.GetBalanceHistory(database))
		r.Get("/api/sync-log", handlers.GetSyncLog(database))
		r.Get("/api/growth", handlers.GetGrowth(database))
		r.Get("/api/net-worth", handlers.GetNetWorth(database))
		r.Put("/api/settings/growth-badge", handlers.SaveGrowthBadge(database))
		r.Post("/api/groups", handlers.CreateGroup(database))
		r.Patch("/api/groups/{id}", handlers.UpdateGroup(database))
		r.Delete("/api/groups/{id}", handlers.DeleteGroup(database))
		r.Post("/api/groups/{id}/members", handlers.AddGroupMember(database))
		r.Delete("/api/groups/{id}/members/{accountId}", handlers.RemoveGroupMember(database))
		r.Post("/api/alerts", handlers.CreateAlert(database))
		r.Get("/api/alerts", handlers.ListAlerts(database))
		r.Put("/api/alerts/{id}", handlers.UpdateAlert(database))
		r.Patch("/api/alerts/{id}", handlers.ToggleAlert(database))
		r.Delete("/api/alerts/{id}", handlers.DeleteAlert(database))
		r.Post("/api/email/config", handlers.SaveEmailConfig(database, jwtSecret))
		r.Get("/api/email/config", handlers.GetEmailConfig(database))
		r.Post("/api/email/test", handlers.TestEmail(database, jwtSecret))
		r.Get("/api/projections/settings", handlers.GetProjectionSettings(database))
		r.Put("/api/projections/settings", handlers.SaveProjectionSettings(database))
		r.Put("/api/projections/income", handlers.SaveIncomeSettings(database))
		r.Get("/api/projections/history", handlers.GetProjectionHistory(database))
	})

	return r
}
