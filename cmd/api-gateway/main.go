// Package main is the entry point for the monolithic Order processing application.
// This single service contains all functionality
//
// 12-Factor App compilance:
//   - I. Codebase: Single codebase tracked in version control
//   - II. Dependencies: Managed via go.mod
//   - III. Config: Configuration via environment variables
//   - VI. Processes: Stateless processes
//   - VII. Port Binding: Self-contained HTTP server
//   - IX. Disposability: Graceful shutdown
//
// Usage:
//
//	go run cmd/api-gateway/main.go
//
// Environment Variables:
//
//	OPS_ENVIRONMENT - Deployment environment (development, staging, production)
//	OPS_SERVER_PORT - HTTP server port (default: 8080)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/hapkiduki/order-go/internal/infrastructure/config"
)

// version is set at build time via ldflags
var version = "dev"

// startTime tracks when the server started for uptime calculations
var startTime = time.Now()

func main() {
	// Load configuration
	cfg := config.MustLoad()

	fmt.Println("Starting Order Processing System (Monolith)\n version:", version, "\n environment:", cfg.App.Environment)

	// Create context that listens for shutdowns signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create Chi router
	r := chi.NewRouter()

	// ============================================================================
	// Middleware stack
	// ============================================================================
	// Order matters! Middleware is executed in the order added.

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.Server.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID", "X-API-Version"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// ============================================================================
	// Routes
	// ============================================================================

	// Health check endpoints (no auth required)
	r.Get("/health", healthHandler())
	//r.Get("/ready", readinessHandler())

	// 404 handler
	r.NotFound(notFoundHandler)

	// 405 handler
	r.MethodNotAllowed(methodNotAllowedHandler)

	// ============================================================================
	// HTTP server
	// ============================================================================

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		fmt.Println("HTTP server starting ", "address", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Errorf("HTTP server failed", "error", err)
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()
	fmt.Println("Shutdown signal received")

	// Create sutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	// Graceful shutdown
	if err := server.Shutdown(shutdownCtx); err != nil {
		fmt.Errorf("Server forced to shutdown", "error", err)
	}
	fmt.Println("Server shutdown complete")

}

// healthHandler returns the health check handler.
func healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "healthy",
			"version": version,
			"uptime":  time.Since(startTime).String(),
		})
	}
}

// readinessHandler returns the readiness check handler.
func readinessHandler() http.HandlerFunc {
	// TODO: verify the database connection
	panic("unimplemented")
}

// notFoundHandler handles 404 responses.
func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error": map[string]string{
			"code":    "NOT_FOUND",
			"message": "The requested resource was not found",
		},
	})
}

// methodNotAllowedHandler handles 405 responses.
func methodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error": map[string]string{
			"code":    "METHOD_NOT_ALLOWED",
			"message": "The requested method is not allowed for this resource",
		},
	})
}
