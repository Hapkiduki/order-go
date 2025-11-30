// Package middleware provides HTTP middleware for the Chi router.
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
	"mime"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/hapkiduki/order-go/internal/application/port"
	"golang.org/x/time/rate"
)

// ContextKey is a custom type for context keys to avoid collisions.
type ContextKey string

const (
	// RequestIDKey is the context key for request IDs.
	RequestIDKey ContextKey = "request_id"

	// RequestIDHeader is the header name for request IDs.
	RequestIDHeader = "X-Request-ID"

	// RealIPKey is the context key for the real client IP.
	RealIPKey ContextKey = "real_ip"
)

// GetRequestID extracts the request ID from the context.
//
// Parameters:
//   - ctx: The request context
//
// Returns:
//   - string: The request ID, or empty string if not found
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// GetRealIP extracts the real client IP from the context.
// Falls back to RemoteAddr if not found in context.
//
// Parameters:
//   - r: The HTTP request
//
// Returns:
//   - string: The real client IP address
func GetRealIP(r *http.Request) string {
	if ip, ok := r.Context().Value(RealIPKey).(string); ok && ip != "" {
		return ip
	}
	// Fallback to RemoteAddr (original behavior)
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	if host != "" {
		return host
	}
	return r.RemoteAddr
}

// RequestID generates a unique request ID for each request.
// The ID is added to the response headers and request context.
//
// Returns:
//   - func(http.Handler) http.Handler: The middleware function
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

// Logger returns a middleware that logs HTTP requests.
// It logs request method, path, status, latency, and client IP.
//
// Parameters:
//   - logger: The logger to use
//
// Returns:
//   - func(http.Handler) http.Handler: The middleware function
func Logger(logger port.Logger) func(http.Handler) http.Handler {
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
				"client_ip", GetRealIP(r),
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
		rw.ResponseWriter.WriteHeader(code)
	}
}

// Write implements http.ResponseWriter.
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

// Recoverer returns a middleware that recovers from panics.
// It logs the panic and returns a 500 Internal Server Error.
//
// Parameters:
//   - logger: The logger to use
//
// Returns:
//   - func(http.Handler) http.Handler: The middleware function
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
					if _, writeErr := w.Write([]byte(`{"success":false,"error":{"code":"INTERNAL_ERROR","message":"An unexpected error occurred"}}`)); writeErr != nil {
						logger.Error("Failed to write error response",
							"request_id", requestID,
							"error", writeErr,
							"path", r.URL.Path,
						)
					}
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimiterConfig contains rate limiter configuration.
type RateLimiterConfig struct {
	// RequestsPerSecond is the number of requests allowed per second
	RequestsPerSecond float64

	// Burst is the maximum burst size
	Burst int

	// KeyFunc extracts the key for rate limiting (e.g., client IP)
	KeyFunc func(*http.Request) string

	// CleanupInterval is how often to clean up inactive limiters
	// Default: 5 minutes
	CleanupInterval time.Duration

	// InactiveTTL is how long a limiter can be inactive before being removed
	// Default: 10 minutes
	InactiveTTL time.Duration
}

// limiterEntry stores a rate limiter with its last access time.
// lastAccess is stored as unix timestamp (int64) for atomic operations.
type limiterEntry struct {
	limiter    *rate.Limiter
	lastAccess int64 // unix timestamp in nanoseconds (atomic)
}

// DefaultRateLimiterConfig returns the default rate limiter configuration.
//
// Returns:
//   - RateLimiterConfig: Default configuration
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		RequestsPerSecond: 10,
		Burst:             20,
		KeyFunc: func(r *http.Request) string {
			return GetRealIP(r)
		},
		CleanupInterval: 5 * time.Minute,
		InactiveTTL:     10 * time.Minute,
	}
}

// RateLimiter returns a middleware that limits request rate per client.
// It uses a token bucket algorithm with per-client buckets.
// The implementation includes automatic cleanup of inactive limiters to prevent memory leaks.
//
// Parameters:
//   - config: Rate limiter configuration
//
// Returns:
//   - func(http.Handler) http.Handler: The middleware function
func RateLimiter(config RateLimiterConfig) func(http.Handler) http.Handler {
	// Set defaults if not provided
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 5 * time.Minute
	}
	if config.InactiveTTL == 0 {
		config.InactiveTTL = 10 * time.Minute
	}

	limiters := make(map[string]*limiterEntry)
	mu := sync.RWMutex{}

	// Context for cleanup goroutine
	// Note: the cleanup goroutine runs for the lifetime of the middleware instance.
	// This is intentional - the middleware is typically created once at startup and
	// lives for the process lifetime. The goroutine will stop when the process exits.
	// If you need to stop cleanup explicitly (e.g., for testing), consider returning
	// the cancel function or using a context passed from the application.
	ctx, cancel := context.WithCancel(context.Background())
	_ = cancel // Intentionally not called - cleanup runs for process lifetime

	// Start cleanup goroutine
	// This goroutine will automatically stop when the process exits
	go func() {
		ticker := time.NewTicker(config.CleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cleanupInactiveLimiters(&mu, limiters, config.InactiveTTL)
			}
		}
	}()

	getLimiter := func(key string) *rate.Limiter {
		now := time.Now().UnixNano()

		mu.RLock()
		entry, exists := limiters[key]
		if exists {
			// Update last access time atomically while holding read lock
			// This prevents race condition where entry could be deleted between
			// releasing read lock and acquiring write lock.
			atomic.StoreInt64(&entry.lastAccess, now)
		}
		mu.RUnlock()

		if exists {
			return entry.limiter
		}

		mu.Lock()
		defer mu.Unlock()

		// Double-check after acquiring write lock
		if entry, exists = limiters[key]; exists {
			atomic.StoreInt64(&entry.lastAccess, now)
			return entry.limiter
		}

		// Create new limiter
		limiter := rate.NewLimiter(rate.Limit(config.RequestsPerSecond), config.Burst)
		limiters[key] = &limiterEntry{
			limiter:    limiter,
			lastAccess: now,
		}

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
				if _, err := w.Write([]byte(`{"success":false,"error":{"code":"RATE_LIMITED","message":"Too many requests, please try again later"}}`)); err != nil {
					// Log write error if logger is available (could be added as parameter)
					// For now, we silently ignore as response writer errors are typically
					// connection issues that can't be recovered
				}
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// cleanupInactiveLimiters removes limiters that haven't been accessed within the TTL period.
func cleanupInactiveLimiters(mu *sync.RWMutex, limiters map[string]*limiterEntry, ttl time.Duration) {
	now := time.Now().UnixNano()
	cutoff := now - ttl.Nanoseconds()

	mu.Lock()
	defer mu.Unlock()

	for key, entry := range limiters {
		// read lastAccess atomically
		lastAccess := atomic.LoadInt64(&entry.lastAccess)
		if lastAccess < cutoff {
			delete(limiters, key)
		}
	}
}

// SecureHeaders returns a middleware that adds security headers.
//
// Returns:
//   - func(http.Handler) http.Handler: The middleware function
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Enable XSS filter

		// Strict Transport Security (if using HTTPS)
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Content Security Policy
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

		// Referrer Policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		next.ServeHTTP(w, r)
	})
}

// APIVersion returns a middleware that adds API version header.
//
// Parameters:
//   - version: API version string
//
// Returns:
//   - func(http.Handler) http.Handler: The middleware function
func APIVersion(version string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-API-Version", version)
			next.ServeHTTP(w, r)
		})
	}
}

// ContentTypeJSON validates that write requests have a valid JSON content type.
// It accepts "application/json" and variants with charset parameters (e.g., "application/json; charset=utf-8").
// Empty Content-Type is allowed (will be set to application/json in response).
//
// Returns:
//   - func(http.Handler) http.Handler: The middleware function
func ContentTypeJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For POST, PUT, PATCH requests, validate JSON content type
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			contentType := r.Header.Get("Content-Type")
			if contentType != "" {
				// Parse media type to handle charset parameters
				mediaType, _, err := mime.ParseMediaType(contentType)
				if err != nil || mediaType != "application/json" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnsupportedMediaType)
					if _, err := w.Write([]byte(`{"success":false,"error":{"code":"UNSUPPORTED_MEDIA_TYPE","message":"Content-Type must be application/json"}}`)); err != nil {
						// Optionally log the error, if a logger is available
						// For now, we just ignore it as there's nothing we can do
					}
					return
				}
			}
		}

		// Set response content type
		w.Header().Set("Content-Type", "application/json")

		next.ServeHTTP(w, r)
	})
}

// RealIP extracts the real client IP from X-Forwarded-For or X-Real-IP headers.
// It handles multiple comma-separated IPs in X-Forwarded-For by taking the first one
// (which is the original client IP). The real IP is stored in the request context
// instead of modifying RemoteAddr to avoid breaking Go's HTTP server assumptions.
//
// Security Note: X-Forwarded-For can be spoofed. In production, validate against
// trusted proxy IPs or use a library that handles proxy headers properly.
//
// Returns:
//   - func(http.Handler) http.Handler: The middleware function
func RealIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var realIP string

		// Try X-Forwarded-For first (can contain multiple IPs: client, proxy1, proxy2)
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// Split by comma and take the first IP (original client IP)
			ips := strings.Split(xff, ",")
			if len(ips) > 0 {
				realIP = strings.TrimSpace(ips[0])
			}
		} else if xri := r.Header.Get("X-Real-IP"); xri != "" {
			// X-Real-IP typically contains a single IP
			realIP = strings.TrimSpace(xri)
		}

		// Validate and store in context instead of modifying RemoteAddr
		if realIP != "" && net.ParseIP(realIP) != nil {
			ctx := context.WithValue(r.Context(), RealIPKey, realIP)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}
