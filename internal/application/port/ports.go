// Package port contains the port interfaces (driven ports) for the application layer.
// Ports define the interfaces that the application layer requires from external
// services like messaging, caching, logging, etc.
//
// In Hexagonal Architecture (Ports & Adapters):
//   - Ports are interfaces that define what the application needs
//   - Adapters are implementations of those interfaces
//   - This enables loose coupling and easy testing/swapping of implementations
//
// SOLID Principles Applied:
//   - Interface Segregation: Small, focused interfaces
//   - Dependency Inversion: Application depends on abstractions
package port

import (
	"context"
	"time"
)

// CacheService defines the interface for caching operations.
// Implementations may use Redis, Memcached, or in-memory caching.
//
// Example usage:
//
//	cache := redis.NewCacheService(client)
//	err := cache.Set(ctx, "order:123", orderData, 300)
//	err := cache.Get(ctx, "order:123", &order)
type CacheService interface {
	// Get retrieves a value from the cache.
	// The value is unmarshaled into the provided destination.
	//
	// Parameters:
	//   - ctx: Context for cancellation and deadlines
	//   - key: The cache key
	//   - dest: Pointer to the destination for unmarshaling
	//
	// Returns:
	//   - error: ErrCacheMiss if key not found, or other error
	Get(ctx context.Context, key string, dest interface{}) error

	// Set stores a value in the cache with the specified TTL.
	//
	// Parameters:
	//   - ctx: Context for cancellation and deadlines
	//   - key: The cache key
	//   - value: The value to cache (will be marshaled)
	//   - ttlSeconds: Time-to-live in seconds (0 for no expiry)
	//
	// Returns:
	//   - error: Any error that occurred during storage
	Set(ctx context.Context, key string, value interface{}, ttlSeconds int) error

	// Delete removes a value from the cache.
	//
	// Parameters:
	//   - ctx: Context for cancellation and deadlines
	//   - key: The cache key
	//
	// Returns:
	//   - error: Any error that occurred during deletion
	Delete(ctx context.Context, key string) error

	// DeletePattern removes all keys matching the pattern.
	//
	// Parameters:
	//   - ctx: Context for cancellation and deadlines
	//   - pattern: The key pattern (e.g., "order:*")
	//
	// Returns:
	//   - error: Any error that occurred during deletion
	DeletePattern(ctx context.Context, pattern string) error

	// Exists checks if a key exists in the cache.
	//
	// Parameters:
	//   - ctx: Context for cancellation and deadlines
	//   - key: The cache key
	//
	// Returns:
	//   - bool: true if key exists
	//   - error: Any error that occurred during check
	Exists(ctx context.Context, key string) (bool, error)

	// SetNX sets a value only if the key doesn't exist (for distributed locks).
	//
	// Parameters:
	//   - ctx: Context for cancellation and deadlines
	//   - key: The cache key
	//   - value: The value to cache
	//   - ttlSeconds: Time-to-live in seconds
	//
	// Returns:
	//   - bool: true if the key was set (didn't exist)
	//   - error: Any error that occurred
	SetNX(ctx context.Context, key string, value interface{}, ttlSeconds int) (bool, error)
}

// Logger defines the interface for structured logging.
// Implementations may use Zap, Logrus, or the standard library.
//
// Example usage:
//
//	logger := zap.NewLogger()
//	logger.Info("Order created", "order_id", orderID, "customer_id", customerID)
type Logger interface {
	// Debug logs a debug message with optional key-value pairs.
	Debug(msg string, keysAndValues ...interface{})

	// Info logs an info message with optional key-value pairs.
	Info(msg string, keysAndValues ...interface{})

	// Warn logs a warning message with optional key-value pairs.
	Warn(msg string, keysAndValues ...interface{})

	// Error logs an error message with optional key-value pairs.
	Error(msg string, keysAndValues ...interface{})

	// With returns a logger with additional context fields.
	With(keysAndValues ...interface{}) Logger

	// WithContext returns a logger with context information (e.g., request ID).
	WithContext(ctx context.Context) Logger
}

// Metrics defines the interface for metrics collection.
// Implementations may use Prometheus, StatsD, or CloudWatch.
type Metrics interface {
	// Counter increments a counter metric.
	Counter(name string, value float64, tags map[string]string)

	// Gauge sets a gauge metric value.
	Gauge(name string, value float64, tags map[string]string)

	// Histogram records a value in a histogram.
	Histogram(name string, value float64, tags map[string]string)

	// Timing records a timing/duration metric.
	Timing(name string, duration time.Duration, tags map[string]string)
}

// Tracer defines the interface for distributed tracing.
// Implementations may use OpenTelemetry, Jaeger, or Zipkin.
type Tracer interface {
	// StartSpan starts a new span for tracing.
	//
	// Parameters:
	//   - ctx: Context for parent span
	//   - operationName: Name of the operation
	//
	// Returns:
	//   - context.Context: Context with the new span
	//   - Span: The created span (must be ended)
	StartSpan(ctx context.Context, operationName string) (context.Context, Span)
}

// Span represents a single operation in a trace.
type Span interface {
	// End ends the span.
	End()

	// SetAttribute sets an attribute on the span.
	SetAttribute(key string, value interface{})

	// SetError marks the span as an error.
	SetError(err error)

	// AddEvent adds an event to the span.
	AddEvent(name string, attributes map[string]interface{})
}
