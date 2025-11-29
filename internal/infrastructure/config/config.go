// Package config provides configuration management for the application.
// It follows the 12-Factor App methodology by loading configuration
// from environment variables and supporting external configuration files.
//
// 12-Factor App Compilance:
// 	 - III. Config: Store config in the environment
// 	 - Configuration is loaded from environment variables
// 	 - Sensitive data (passwords, keys) only via environment
// 	 - No config files checked into version control

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
}

// AppConfig contains application-level configuration.
type AppConfig struct {
	// Name of the application
	Name string `mapstructure:"name"`

	// Environment the application is running in (e.g., development, staging, production)
	Environment string `mapstructure:"environment"`

	// Version of the application
	Version string `mapstructure:"version"`

	// Debug mode flag
	Debug bool `mapstructure:"debug"`
}

// ServerConfig contains HTTP server configuration.
type ServerConfig struct {
	// Host is the server bind address
	Host string `mapstructure:"host"`

	// Port is the server port
	Port int `mapstructure:"port"`

	// ReadTimeout is the maximum duration for reading the entire request, including the body
	ReadTimeout time.Duration `mapstructure:"read_timeout"`

	// WriteTimeout is the maximum duration before timing out writes of the response
	WriteTimeout time.Duration `mapstructure:"write_timeout"`

	// IdleTimeout is the maximum amount of time to wait for the next request when keep-alives are enabled
	IdleTimeout time.Duration `mapstructure:"idle_timeout"`

	// ShutdownTimeout is the maximum duration for graceful server shutdown
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`

	// MaxRequestSize is the maximun allowed request body size
	MaxRequestSize int64 `mapstructure:"max_request_size"`

	// CORSAllowedOrigins is a list of allowed origins for CORS
	CORSAllowedOrigins []string `mapstructure:"cors_allowed_origins"`
}

// Load loads the configuration from environment variables and config files.
// It follows this precedence (higest to lowest):
//  1. Environment variables
//  2. Config file (if provided)
//  3. Default values
//
// Returns:
//   - *Config: The loaded configuration
//   - error: Any error encountered during loading
func Load() (*Config, error) {
	v := viper.New()

	// Set default values
	setDefaults(v)

	// Set config file settings
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./configs")
	v.AddConfigPath("/etc/order-go")

	// Read config file if exists
	if err := v.ReadInConfig(); err != nil {
		// If the error is not "file not found", return the error
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

	// Unmarshal into Config struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default configuration values.
func setDefaults(v *viper.Viper) {
	// App defaults
	v.SetDefault("app.name", "order-go")
	v.SetDefault("app.environment", "development")
	v.SetDefault("app.version", "1.0.0")
	v.SetDefault("app.debug", false)

	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", 15*time.Second)
	v.SetDefault("server.write_timeout", 15*time.Second)
	v.SetDefault("server.idle_timeout", 60*time.Second)
	v.SetDefault("server.shutdown_timeout", 30*time.Second)
	v.SetDefault("server.max_request_size", 10<<20)            // 10MB
	v.SetDefault("server.cors_allowed_origins", []string{"*"}) // Allow all origins by default
}

// bindEnvVars binds specific environment variables to configuration keys.
func bindEnvVars(v *viper.Viper) {
	// These are explicity bound for clarity
	v.BindEnv("app.environment", "OPS_ENVIRONMENT")
	v.BindEnv("server.port", "PORT") // Common convention
}

// loadSensitiveConfig loads sensitive configuration from environment variables.
// This ensures passwords and secrets are never in config files.
func loadSensitiveConfig(cfg *Config) {
	// TODO: Implement loading of sensitive data for database, redis, rabbitmq, etc.
}

// MustLoad loads the configuration and panics on error.
// Use this in application entry points where configuration is required.
//
// Returns:
//   - *Config: The loaded configuration
func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
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
