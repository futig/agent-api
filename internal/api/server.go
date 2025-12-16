package api

import (
	"net/http"
	"time"

	"github.com/futig/agent-backend/internal/api/docs"
	"github.com/futig/agent-backend/internal/api/middleware"
	projectapi "github.com/futig/agent-backend/internal/api/project"
	sessionapi "github.com/futig/agent-backend/internal/api/session"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// SetupRouter creates and configures the HTTP router
func SetupRouter(projectHandler *projectapi.Handler, sessionHandler *sessionapi.Handler, logger *zap.Logger) http.Handler {
	r := chi.NewRouter()

	// Middleware stack
	r.Use(chimiddleware.Recoverer)                 // Recover from panics
	r.Use(chimiddleware.RequestID)                 // Add request ID
	r.Use(middleware.Logger(logger))               // Log requests
	r.Use(middleware.CORS)                         // Handle CORS
	r.Use(chimiddleware.Timeout(60 * time.Second)) // Default timeout

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})

	// Swagger documentation endpoints
	docs.RegisterRoutes(r)

	// Register routes
	projectapi.RegisterRoutes(r, projectHandler)
	sessionapi.RegisterRoutes(r, sessionHandler)

	return r
}
