// Package middleware provides HTTP middleware for Chi router.
// Middleware components handle cross-cutting concerns like logging, authentication,
// rate limiting, and request tracing.
//
// Chi Middleware Philosophy:
//   - Uses standard net/http handlers
//   - Composable middleware chain
//   - Context-based request scoping
//   - Compatible with any net/http middleware
package middleware

import (
	"context"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hapkiduki/order-go/internal/application/port"
	"golang.org/x/time/rate"
)

// ContextKey is a custom type for context keys to avoid collisions.
type ContextKey string

const (
	// RequestIDKey is the context key for the request ID.
	RequestIDKey ContextKey = "request_id"

	// RequestIDHeader is the header name for request IDs.
	RequestIDHeader = "X-Request-ID"
)

// GetRequestID extracts the request ID from the context.
//
// Parameters:
//   - ctx: the request context
//
// Returns:
//   - string: the request ID, or empty string if not found
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// RequestID generates a unique request ID for each request.
// The ID is added to the response headers and request context.
//
// Returns:
//   - func(http.Handler) http.Handler: the middleware function
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if request already has an ID (e.g., from a gateway)
		requestID := r.Header.Get(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set request ID in context and response header
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
		w.Header().Set(RequestIDHeader, requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Logger returns a middleware that logs HTTP request.
// It logs request method, path, status, latency, and client IP.
//
// Parameters:
//   - logger: The logger to use
//
// Returns:
//   - func(http.Handler) http.Handler: the middleware function
func Logger(logger port.Logger) func(w http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Process request
			next.ServeHTTP(ww, r)

			// Calculate latency
			latency := time.Since(start)

			// Get request ID from context
			requestID := GetRequestID(r.Context())

			// Log request details
			logger.Info("HTTP Request",
				"request_id", requestID,
				"method", r.Method,
				"path", r.URL.Path,
				"query", r.URL.RawQuery,
				"status", ww.statusCode,
				"latency_ms", latency.Milliseconds(),
				"client_ip", r.RemoteAddr,
				"user_agent", r.UserAgent(),
			)
		})

	}
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// WriteHeader captures the status code.
func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

// Write implements http.ResponseWriter.
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

// Recoverer returns a middleware that recovers from panics.
// It logs the panic and returns a 500 Internal Server Error response.
//
// Parameters:
//   - logger: The logger to use
//
// Returns:
//   - func(http.Handler) http.Handler: the middleware function
func Recoverer(logger port.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					requestID := GetRequestID(r.Context())

					logger.Error("Panic recovered",
						"request_id", requestID,
						"error", err,
						"path", r.URL.Path,
						"stack", string(debug.Stack()),
					)

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"success": false, "error": {"code": "INTERNAL_ERROR", "message": "An unexpected error occurred"}}`))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimiterConfig contains rate limiter configuration.
type RateLimiterConfig struct {
	// RequestedPerSecond is the number of requests allowed per second.
	RequestedPerSecond float64

	// Burst is the maximum burst size.
	Burst int

	// KeyFunc extracts the key for rate limiting (e.g., client IP).
	KeyFunc func(*http.Request) string
}

// DefaultRateLimiterConfig returns the default rate limiter configuration.
//
// Returns:
//   - RateLimiterConfig: default configuration
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		RequestedPerSecond: 10,
		Burst:              20,
		KeyFunc: func(r *http.Request) string {
			return r.RemoteAddr
		},
	}
}

// RateLimiter returns a middleware that limits request rate per client.
// It uses a token bucket algorithm with per-client buckets.
//
// Parameters:
//   - config: Rate limiter configuration
//
// Returns:
//   - func(http.Handler) http.Handler: the middleware function
func RateLimiter(config RateLimiterConfig) func(http.Handler) http.Handler {
	limiters := make(map[string]*rate.Limiter)
	mu := sync.RWMutex{}

	getLimiter := func(key string) *rate.Limiter {
		mu.RLock()
		limiter, exists := limiters[key]
		mu.RUnlock()

		if exists {
			return limiter
		}

		mu.Lock()
		defer mu.Unlock()

		// Double-check after acquiring write lock
		if limiter, exists = limiters[key]; exists {
			return limiter
		}

		limiter = rate.NewLimiter(rate.Limit(config.RequestedPerSecond), config.Burst)
		limiters[key] = limiter
		return limiter
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := config.KeyFunc(r)
			limiter := getLimiter(key)

			if !limiter.Allow() {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"success": false, "error": {"code": "RATE_LIMITED", "message": "Too many requests, please try again later"}}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SecureHeaders returns a middleware that adds security headers.
//
// Returns:
//   - func(http.Handler) http.Handler: the middleware function
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Enable XSS filter
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Strict transport security (if using HTTPS)
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Content Security Policy
		w.Header().Set("Content-Security-Policy", "default-src 'self'")

		// Referrer Policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		next.ServeHTTP(w, r)
	})
}

// APIVersion returns a middleware that adds API version header.
//
// Parameters:
//   - version: The API version string
//
// Returns:
//   - func(http.Handler) http.Handler: the middleware function
func APIVersion(version string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-API-Version", version)
			next.ServeHTTP(w, r)
		})
	}
}

// ContentTypeJSON ensure request have JSON content type for write operations.
//
// Returns:
//   - func(http.Handler) http.Handler: the middleware function
func ContentTypeJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For POST, PUT, PATCH request, ensure JSON content type
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			if r.Header.Get("Content-Type") != "application/json" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnsupportedMediaType)
				w.Write([]byte(`{"success": false, "error": {"code": "UNSUPPORTED_MEDIA_TYPE", "message": "Content-Type must be application/json"}}`))
				return
			}
		}
		// Set response content type
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// Timeout returns a middleware that enforces a request timeout
//
// Parameters:
//   - timeout: Maximum request duration
//
// Returns:
//   - func(http.Handler) http.Handler: the middleware function
func Timeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// create a channel to signal completion
			done := make(chan struct{})

			go func() {
				next.ServeHTTP(w, r.WithContext(ctx))
				close(done)
			}()

			select {
			case <-done:
				return
			case <-ctx.Done():
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusGatewayTimeout)
				w.Write([]byte(`{"success": false, "error": {"code": "TIMEOUT", "message": "Request timed out"}}`))
			}
		})
	}
}

// RealIP extracts the real client IP from X-Forwarded-For or X-Real-IP headers.
//
// Returns:
//   - func(http.Handler) http.Handler: the middleware function
func RealIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// try X-Forwarded-For first
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			r.RemoteAddr = xff
		} else if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
			r.RemoteAddr = xrip
		}

		next.ServeHTTP(w, r)
	})
}
