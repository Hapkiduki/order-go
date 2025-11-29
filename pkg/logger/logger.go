// Package logger provides structured logging utilities.
// It wraps the zap logger with a simplified interface that follows
// the 12-Factor App logging principles (logs as event streams).
//
// 12-Factor App compliance:
//   - XI. Logs: Treat logs as event streams
//   - Output to stdout, no log file management
//   - Structured logging format (JSON) for easy parsing
package logger

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// contextKey is a custom type for context keys.
type contextKey string

const (
	// RequestIDKey is the context key for the request ID.
	RequestIDKey contextKey = "request_id"

	// UserIDKey is the context key for the user ID.
	UserIDKey contextKey = "user_id"
)

// Logger is the application logger interface implementation.
type Logger struct {
	zap    *zap.Logger
	sugar  *zap.SugaredLogger
	fields []interface{}
}

// Config contains logger configuration.
type Config struct {
	// Level is the minimun log level (debug, info, warn, error).
	Level string

	// Format is the output format (json, console).
	Format string

	// Development enables development mode (more verbose)
	Development bool
}

// DefaultConfig returns the default logger configuration.
//
// Returns:
//   - Config: default logger configuration
func DefaultConfig() Config {
	return Config{
		Level:       "info",
		Format:      "json",
		Development: false,
	}
}

// New creates a new Logger with the given configuration.
//
// Parameters:
//   - cfg: Logger configuration
//
// Returns:
//   - *Logger: configured logger instance
//   - error: Any error during initialization
func new(cfg Config) (*Logger, error) {
	// Parse log level
	level := zapcore.InfoLevel
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		return nil, err
	}

	// configure encoder
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.LowercaseColorLevelEncoder

	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Create core
	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		level,
	)

	// Build logger
	opts := []zap.Option{
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	}

	if cfg.Development {
		opts = append(opts, zap.Development())
	}

	zapLogger := zap.New(core, opts...)

	return &Logger{
		zap:   zapLogger,
		sugar: zapLogger.Sugar(),
	}, nil
}

// MustNew creates a new Logger and panics on error.
//
// Parameters:
//   - cfg: Logger configuration
//
// Returns:
//   - *Logger: configured logger instance
func MustNew(cfg Config) *Logger {
	logger, err := new(cfg)
	if err != nil {
		panic(err)
	}
	return logger
}

// Debug logs a debug message with optional key-value pairs.
//
// Parameters:
//   - msg: the log message
//   - keysAndValues: optional key-value pairs for structured logging
func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	l.sugar.Debugw(msg, append(l.fields, keysAndValues...)...)
}

// Info logs an info message with optional key-value pairs.
//
// Parameters:
//   - msg: the log message
//   - keysAndValues: optional key-value pairs for structured logging
func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	l.sugar.Infow(msg, append(l.fields, keysAndValues...)...)
}

// Warn logs a warning message with optional key-value pairs.
//
// Parameters:
//   - msg: the log message
//   - keysAndValues: optional key-value pairs for structured logging
func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	l.sugar.Warnw(msg, append(l.fields, keysAndValues...)...)
}

// Error logs an error message with optional key-value pairs.
//
// Parameters:
//   - msg: the log message
//   - keysAndValues: optional key-value pairs for structured logging
func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	l.sugar.Errorw(msg, append(l.fields, keysAndValues...)...)
}

// Fatal logs a fatal message and exits the program.
//
// Parameters:
//   - msg: the log message
//   - keysAndValues: optional key-value pairs for structured logging
func (l *Logger) Fatal(msg string, keysAndValues ...interface{}) {
	l.sugar.Fatalw(msg, append(l.fields, keysAndValues...)...)
}

// With return a logger with additional context fields.
// These fields will be included in all subsequent log entries.
//
// Parameters:
//   - keysAndValues: key-value pairs to add
//
// Returns:
//   - Logger: new logger with additional fields
func (l *Logger) With(keysAndValues ...interface{}) *Logger {
	return &Logger{
		zap:    l.zap,
		sugar:  l.sugar,
		fields: append(l.fields, keysAndValues...),
	}
}

// WithContext return a logger with context information (e.g., request ID, user ID, etc.).
//
// Parameters:
//   - ctx: the context to extract values from
//
// Returns:
//   - Logger: new logger with context fields
func (l *Logger) WithContext(ctx context.Context) *Logger {
	fields := make([]any, 0, len(l.fields)+4)
	fields = append(fields, l.fields...)

	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		fields = append(fields, "request_id", requestID)
	}

	if userID := ctx.Value(UserIDKey); userID != nil {
		fields = append(fields, "user_id", userID)
	}

	return &Logger{
		zap:    l.zap,
		sugar:  l.sugar,
		fields: fields,
	}
}

// Sync flushes any buffered log entries.
// Should be called before application exit.
//
// Returns:
//   - error: Any error during sync
func (l *Logger) Sync() error {
	return l.zap.Sync()
}

// Named returns a named logger
//
// Parameters:
//   - name: The logger name (will be added to log output)
//
// Returns:
//   - *Logger: A named logger
func (l *Logger) Named(name string) *Logger {
	return &Logger{
		zap:    l.zap.Named(name),
		sugar:  l.zap.Named(name).Sugar(),
		fields: l.fields,
	}
}

// ZapLogger returns the underlying zap.Logger instance.
// Use this when you need direct access to zap features.
//
// Returns:
//   - *zap.Logger: the underlying zap logger
func (l *Logger) ZapLogger() *zap.Logger {
	return l.zap
}

// GLobal logger instance
var globalLogger *Logger

// init initializes the global logger with default settings.
func init() {
	var err error
	globalLogger, err = new(DefaultConfig())
	if err != nil {
		panic(err)
	}
}

// SetGlobal sets the global logger instance.
//
// Parameters:
//   - logger: The logger instance to set as global
func SetGlobal(logger *Logger) {
	globalLogger = logger
}

// Global returns the global logger instance.
//
// Returns:
//   - *Logger: The global logger instance
func Global() *Logger {
	return globalLogger
}

// Debug logs a debug message using the global logger.
func Debug(msg string, keysAndValues ...any) {
	globalLogger.Debug(msg, keysAndValues...)
}

// Info logs an info message using the global logger.
func Info(msg string, keysAndValues ...any) {
	globalLogger.Info(msg, keysAndValues...)
}

// Warn logs a warning message using the global logger.
func Warn(msg string, keysAndValues ...any) {
	globalLogger.Warn(msg, keysAndValues...)
}

// Error logs an error message using the global logger.
func Error(msg string, keysAndValues ...any) {
	globalLogger.Error(msg, keysAndValues...)
}

// Fatal logs a fatal message and exits the program using the global logger.
func Fatal(msg string, keysAndValues ...any) {
	globalLogger.Fatal(msg, keysAndValues...)
}
