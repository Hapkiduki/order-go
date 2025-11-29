// Package port contains the port interfaces (driven ports) for the application layer.
// Ports define the interfaces that the application layer requires from external
// services like messaging, caching, logging, etc.
//
// In Hexagonal Architecture (ports & adapters):
//   - Ports are interfaces that define what the application needs.
//   - Adapters are implementations of these interfaces
//   - this enables loose coupling and easy testing/swapping of implementations.
//
// SOLID Principles applied:
//   - Interface Segregation: small, focused interfaces
//   - Dependency Inversion: Application depends on abstractions
package port

import (
	"context"
	"time"
)

// Logger defines the interface for structured logging.
// Implementation may use zap, logrus, or the standard library.
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

	// With return a logger with additional context fields.
	With(keysAndValues ...interface{}) Logger

	// WithContext return a logger with context information (e.g., request ID).
	WithContext(ctx context.Context) Logger
}

// Metrics defines the interface for recording application metrics.
// Implementation may use Prometheus, StatsD, or CloudWatch.
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
// Implementation may use OpenTelemetry, Jaeger, or Zipkin.
type Tracer interface {
	// StartSpan starts a new span for tracing.
	//
	// Parameters:
	//   - ctx: the context for parent span
	//   - operationName: the name of the operation being traced
	//
	// Returns:
	//   - context.Context: the new context containing the span
	//   - Span: the created span (must be ended)
	StartSpan(ctx context.Context, operationName string) (context.Context, Span)
}

// Span represents a single operation in a trace.
type Span interface {
	// End ends the span.
	End()

	// SetAttribute sets an attribute on the span.
	SetAttribute(key string, value interface{})

	// SetError marks the span with an error.
	SetError(err error)

	// AddEvent adds an event to the span.
	AddEvent(name string, attributes map[string]interface{})
}
