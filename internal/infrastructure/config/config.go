// Package config provides application configuration management.
// It follows the 12-Factor App methodology by loading configuration
// from environment variables and supporting external configuration files.
//
// 12-Factor App Compliance:
//   - III. Config: Store config in the environment
//   - Configuration is loaded from environment variables
//   - Sensitive data (passwords, keys) only via environment
//   - No config files checked into version control
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
// All fields are populated from environment variables or config files.
type Config struct {
	// App contains application-level configuration
	App AppConfig `mapstructure:"app"`

	// Server contains HTTP server configuration
	Server ServerConfig `mapstructure:"server"`

	// Database contains PostgreSQL configuration
	Database DatabaseConfig `mapstructure:"database"`

	// Redis contains Redis cache configuration
	Redis RedisConfig `mapstructure:"redis"`

	// RabbitMQ contains message queue configuration
	RabbitMQ RabbitMQConfig `mapstructure:"rabbitmq"`

	// Log contains logging configuration
	Log LogConfig `mapstructure:"log"`

	// Metrics contains metrics configuration
	Metrics MetricsConfig `mapstructure:"metrics"`

	// Tracing contains distributed tracing configuration
	Tracing TracingConfig `mapstructure:"tracing"`
}

// AppConfig contains application-level configuration.
type AppConfig struct {
	// Name is the application name
	Name string `mapstructure:"name"`

	// Environment is the deployment environment (development, staging, production)
	Environment string `mapstructure:"environment"`

	// Version is the application version
	Version string `mapstructure:"version"`

	// Debug enables debug mode
	Debug bool `mapstructure:"debug"`
}

// ServerConfig contains HTTP server configuration.
type ServerConfig struct {
	// Host is the server bind address
	Host string `mapstructure:"host"`

	// Port is the server port
	Port int `mapstructure:"port"`

	// ReadTimeout is the maximum duration for reading the entire request
	ReadTimeout time.Duration `mapstructure:"read_timeout"`

	// WriteTimeout is the maximum duration before timing out writes
	WriteTimeout time.Duration `mapstructure:"write_timeout"`

	// IdleTimeout is the maximum time to wait for the next request
	IdleTimeout time.Duration `mapstructure:"idle_timeout"`

	// ShutdownTimeout is the maximum time to wait for graceful shutdown
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`

	// MaxRequestSize is the maximum allowed request body size
	MaxRequestSize int64 `mapstructure:"max_request_size"`

	// CORSAllowedOrigins is the list of allowed CORS origins
	CORSAllowedOrigins []string `mapstructure:"cors_allowed_origins"`
}

// DatabaseConfig contains PostgreSQL configuration.
type DatabaseConfig struct {
	// Host is the database host
	Host string `mapstructure:"host"`

	// Port is the database port
	Port int `mapstructure:"port"`

	// User is the database user
	User string `mapstructure:"user"`

	// Password is the database password (from environment)
	Password string `mapstructure:"password"`

	// Database is the database name
	Database string `mapstructure:"database"`

	// SSLMode is the SSL mode (disable, require, verify-ca, verify-full)
	SSLMode string `mapstructure:"ssl_mode"`

	// MaxOpenConns is the maximum number of open connections
	MaxOpenConns int `mapstructure:"max_open_conns"`

	// MaxIdleConns is the maximum number of idle connections
	MaxIdleConns int `mapstructure:"max_idle_conns"`

	// ConnMaxLifetime is the maximum lifetime of a connection
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`

	// ConnMaxIdleTime is the maximum idle time of a connection
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
}

// DSN returns the PostgreSQL connection string.
func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
		c.SSLMode,
	)
}

// RedisConfig contains Redis configuration.
type RedisConfig struct {
	// Host is the Redis host
	Host string `mapstructure:"host"`

	// Port is the Redis port
	Port int `mapstructure:"port"`

	// Password is the Redis password (from environment)
	Password string `mapstructure:"password"`

	// DB is the Redis database number
	DB int `mapstructure:"db"`

	// PoolSize is the connection pool size
	PoolSize int `mapstructure:"pool_size"`

	// KeyPrefix is the prefix for all cache keys
	KeyPrefix string `mapstructure:"key_prefix"`
}

// RabbitMQConfig contains RabbitMQ configuration.
type RabbitMQConfig struct {
	// Host is the RabbitMQ host
	Host string `mapstructure:"host"`

	// Port is the RabbitMQ port
	Port int `mapstructure:"port"`

	// User is the RabbitMQ user
	User string `mapstructure:"user"`

	// Password is the RabbitMQ password (from environment)
	Password string `mapstructure:"password"`

	// VHost is the RabbitMQ virtual host
	VHost string `mapstructure:"vhost"`

	// Exchange is the default exchange name
	Exchange string `mapstructure:"exchange"`

	// ExchangeType is the exchange type (direct, fanout, topic)
	ExchangeType string `mapstructure:"exchange_type"`
}

// URL returns the RabbitMQ connection URL.
func (c RabbitMQConfig) URL() string {
	return fmt.Sprintf(
		"amqp://%s:%s@%s:%d/%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.VHost,
	)
}

// LogConfig contains logging configuration.
type LogConfig struct {
	// Level is the log level (debug, info, warn, error)
	Level string `mapstructure:"level"`

	// Format is the log format (json, console)
	Format string `mapstructure:"format"`

	// Output is the log output (stdout, stderr, file path)
	Output string `mapstructure:"output"`
}

// MetricsConfig contains metrics configuration.
type MetricsConfig struct {
	// Enabled indicates if metrics are enabled
	Enabled bool `mapstructure:"enabled"`

	// Port is the metrics server port
	Port int `mapstructure:"port"`

	// Path is the metrics endpoint path
	Path string `mapstructure:"path"`
}

// TracingConfig contains distributed tracing configuration.
type TracingConfig struct {
	// Enabled indicates if tracing is enabled
	Enabled bool `mapstructure:"enabled"`

	// ServiceName is the service name for traces
	ServiceName string `mapstructure:"service_name"`

	// Endpoint is the tracing collector endpoint
	Endpoint string `mapstructure:"endpoint"`

	// SamplingRate is the trace sampling rate (0.0 to 1.0)
	SamplingRate float64 `mapstructure:"sampling_rate"`
}

// Load loads configuration from environment variables and config files.
// It follows this precedence (highest to lowest):
//  1. Environment variables
//  2. Config file
//  3. Default values
//
// Returns:
//   - *Config: The loaded configuration
//   - error: Any error that occurred during loading
func Load() (*Config, error) {
	v := viper.New()

	// Set default values
	setDefaults(v)

	// Set config file settings
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./configs")
	v.AddConfigPath("/etc/order-processing-system")

	// Read config file if exists
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is OK, we'll use env vars and defaults
	}

	// Read environment variables
	v.SetEnvPrefix("OPS") // Order Processing System
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Bind specific environment variables
	bindEnvVars(v)

	// Unmarshal into config struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Load sensitive values directly from environment
	loadSensitiveConfig(&cfg)

	return &cfg, nil
}

// setDefaults sets default configuration values.
func setDefaults(v *viper.Viper) {
	// App defaults
	v.SetDefault("app.name", "order-processing-system")
	v.SetDefault("app.environment", "development")
	v.SetDefault("app.version", "1.0.0")
	v.SetDefault("app.debug", false)

	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", 30*time.Second)
	v.SetDefault("server.write_timeout", 30*time.Second)
	v.SetDefault("server.idle_timeout", 120*time.Second)
	v.SetDefault("server.shutdown_timeout", 30*time.Second)
	v.SetDefault("server.max_request_size", 10<<20) // 10 MB
	v.SetDefault("server.cors_allowed_origins", []string{"*"})

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.database", "orders")
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", 5*time.Minute)
	v.SetDefault("database.conn_max_idle_time", 5*time.Minute)

	// Redis defaults
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 10)
	v.SetDefault("redis.key_prefix", "ops:")

	// RabbitMQ defaults
	v.SetDefault("rabbitmq.host", "localhost")
	v.SetDefault("rabbitmq.port", 5672)
	v.SetDefault("rabbitmq.user", "guest")
	v.SetDefault("rabbitmq.vhost", "/")
	v.SetDefault("rabbitmq.exchange", "order_events")
	v.SetDefault("rabbitmq.exchange_type", "topic")

	// Log defaults
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("log.output", "stdout")

	// Metrics defaults
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.port", 9090)
	v.SetDefault("metrics.path", "/metrics")

	// Tracing defaults
	v.SetDefault("tracing.enabled", false)
	v.SetDefault("tracing.service_name", "order-processing-system")
	v.SetDefault("tracing.sampling_rate", 0.1)
}

// bindEnvVars binds specific environment variables to config keys.
func bindEnvVars(v *viper.Viper) {
	// These are explicitly bound for clarity
	v.BindEnv("app.environment", "OPS_ENVIRONMENT")
	v.BindEnv("server.port", "PORT") // Common convention
	v.BindEnv("database.host", "OPS_DATABASE_HOST", "DATABASE_HOST")
	v.BindEnv("database.port", "OPS_DATABASE_PORT", "DATABASE_PORT")
	v.BindEnv("database.user", "OPS_DATABASE_USER", "DATABASE_USER")
	v.BindEnv("database.database", "OPS_DATABASE_NAME", "DATABASE_NAME")
	v.BindEnv("redis.host", "OPS_REDIS_HOST", "REDIS_HOST")
	v.BindEnv("rabbitmq.host", "OPS_RABBITMQ_HOST", "RABBITMQ_HOST")
}

// loadSensitiveConfig loads sensitive configuration from environment variables.
// This ensures passwords and secrets are never in config files.
func loadSensitiveConfig(cfg *Config) {
	// Database password
	if pwd := os.Getenv("OPS_DATABASE_PASSWORD"); pwd != "" {
		cfg.Database.Password = pwd
	} else if pwd := os.Getenv("DATABASE_PASSWORD"); pwd != "" {
		cfg.Database.Password = pwd
	}

	// Redis password
	if pwd := os.Getenv("OPS_REDIS_PASSWORD"); pwd != "" {
		cfg.Redis.Password = pwd
	} else if pwd := os.Getenv("REDIS_PASSWORD"); pwd != "" {
		cfg.Redis.Password = pwd
	}

	// RabbitMQ password
	if pwd := os.Getenv("OPS_RABBITMQ_PASSWORD"); pwd != "" {
		cfg.RabbitMQ.Password = pwd
	} else if pwd := os.Getenv("RABBITMQ_PASSWORD"); pwd != "" {
		cfg.RabbitMQ.Password = pwd
	} else {
		cfg.RabbitMQ.Password = "guest" // Default for development
	}
}

// MustLoad loads configuration and panics on error.
// Use this in application entry points where configuration is required.
//
// Returns:
//   - *Config: The loaded configuration
func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load configuration: %v", err))
	}
	return cfg
}

// GetEnv gets an environment variable with a default value.
//
// Parameters:
//   - key: Environment variable name
//   - defaultValue: Default value if not set
//
// Returns:
//   - string: The environment variable value or default
func GetEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// GetEnvInt gets an integer environment variable with a default value.
//
// Parameters:
//   - key: Environment variable name
//   - defaultValue: Default value if not set or invalid
//
// Returns:
//   - int: The environment variable value or default
func GetEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetEnvBool gets a boolean environment variable with a default value.
//
// Parameters:
//   - key: Environment variable name
//   - defaultValue: Default value if not set or invalid
//
// Returns:
//   - bool: The environment variable value or default
func GetEnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
