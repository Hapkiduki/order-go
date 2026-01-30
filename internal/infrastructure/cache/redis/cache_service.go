// Package redis provides Redis implementations of caching interfaces.
// It uses the go-redis client for high-performance cache operations.
//
// Features:
//   - JSON serialization for complex objects
//   - Pattern-based key deletion
//   - Distributed locking support
//   - Connection pooling
package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Common cache errors.
var (
	ErrCacheMiss       = errors.New("cache miss: key not found")
	ErrSerializationFailed = errors.New("failed to serialize value")
	ErrDeserializationFailed = errors.New("failed to deserialize value")
)

// CacheService is the Redis implementation of the CacheService interface.
// It provides caching operations with JSON serialization for complex objects.
type CacheService struct {
	client *redis.Client
	prefix string // Key prefix for namespacing
}

// Config contains Redis connection configuration.
type Config struct {
	// Host is the Redis server host
	Host string

	// Port is the Redis server port
	Port int

	// Password is the Redis password (empty if none)
	Password string

	// DB is the Redis database number
	DB int

	// PoolSize is the connection pool size
	PoolSize int

	// KeyPrefix is the prefix for all keys
	KeyPrefix string
}

// NewCacheService creates a new CacheService with the given configuration.
//
// Parameters:
//   - cfg: Redis configuration
//
// Returns:
//   - *CacheService: The cache service instance
//   - error: Any error that occurred during connection
func NewCacheService(cfg Config) (*CacheService, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	prefix := cfg.KeyPrefix
	if prefix == "" {
		prefix = "ops:" // Order Processing System
	}

	return &CacheService{
		client: client,
		prefix: prefix,
	}, nil
}

// NewCacheServiceFromClient creates a CacheService from an existing Redis client.
// Useful for testing with mock clients.
//
// Parameters:
//   - client: Existing Redis client
//   - prefix: Key prefix
//
// Returns:
//   - *CacheService: The cache service instance
func NewCacheServiceFromClient(client *redis.Client, prefix string) *CacheService {
	return &CacheService{
		client: client,
		prefix: prefix,
	}
}

// fullKey returns the full key with prefix.
func (c *CacheService) fullKey(key string) string {
	return c.prefix + key
}

// Get retrieves a value from the cache.
// The value is unmarshaled from JSON into the provided destination.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - key: The cache key
//   - dest: Pointer to the destination for unmarshaling
//
// Returns:
//   - error: ErrCacheMiss if key not found, or other error
func (c *CacheService) Get(ctx context.Context, key string, dest interface{}) error {
	fullKey := c.fullKey(key)

	val, err := c.client.Get(ctx, fullKey).Bytes()
	if errors.Is(err, redis.Nil) {
		return ErrCacheMiss
	}
	if err != nil {
		return fmt.Errorf("failed to get key %s: %w", key, err)
	}

	if err := json.Unmarshal(val, dest); err != nil {
		return fmt.Errorf("%w: %v", ErrDeserializationFailed, err)
	}

	return nil
}

// Set stores a value in the cache with the specified TTL.
// The value is marshaled to JSON before storage.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - key: The cache key
//   - value: The value to cache (will be marshaled to JSON)
//   - ttlSeconds: Time-to-live in seconds (0 for no expiry)
//
// Returns:
//   - error: Any error that occurred during storage
func (c *CacheService) Set(ctx context.Context, key string, value interface{}, ttlSeconds int) error {
	fullKey := c.fullKey(key)

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSerializationFailed, err)
	}

	var ttl time.Duration
	if ttlSeconds > 0 {
		ttl = time.Duration(ttlSeconds) * time.Second
	}

	if err := c.client.Set(ctx, fullKey, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set key %s: %w", key, err)
	}

	return nil
}

// Delete removes a value from the cache.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - key: The cache key
//
// Returns:
//   - error: Any error that occurred during deletion
func (c *CacheService) Delete(ctx context.Context, key string) error {
	fullKey := c.fullKey(key)

	if err := c.client.Del(ctx, fullKey).Err(); err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}

	return nil
}

// DeletePattern removes all keys matching the pattern.
// Uses SCAN for safety in production environments.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - pattern: The key pattern (e.g., "order:*")
//
// Returns:
//   - error: Any error that occurred during deletion
func (c *CacheService) DeletePattern(ctx context.Context, pattern string) error {
	fullPattern := c.fullKey(pattern)

	var cursor uint64
	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, fullPattern, 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan keys: %w", err)
		}

		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("failed to delete keys: %w", err)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

// Exists checks if a key exists in the cache.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - key: The cache key
//
// Returns:
//   - bool: true if key exists
//   - error: Any error that occurred during check
func (c *CacheService) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := c.fullKey(key)

	count, err := c.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence of key %s: %w", key, err)
	}

	return count > 0, nil
}

// SetNX sets a value only if the key doesn't exist.
// Useful for implementing distributed locks.
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
func (c *CacheService) SetNX(ctx context.Context, key string, value interface{}, ttlSeconds int) (bool, error) {
	fullKey := c.fullKey(key)

	data, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrSerializationFailed, err)
	}

	ttl := time.Duration(ttlSeconds) * time.Second

	result, err := c.client.SetNX(ctx, fullKey, data, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to setnx key %s: %w", key, err)
	}

	return result, nil
}

// Increment atomically increments a counter.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - key: The cache key
//   - delta: The amount to increment by
//
// Returns:
//   - int64: The new value after increment
//   - error: Any error that occurred
func (c *CacheService) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	fullKey := c.fullKey(key)

	result, err := c.client.IncrBy(ctx, fullKey, delta).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment key %s: %w", key, err)
	}

	return result, nil
}

// SetTTL updates the TTL of an existing key.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - key: The cache key
//   - ttlSeconds: New TTL in seconds
//
// Returns:
//   - error: Any error that occurred
func (c *CacheService) SetTTL(ctx context.Context, key string, ttlSeconds int) error {
	fullKey := c.fullKey(key)

	ttl := time.Duration(ttlSeconds) * time.Second

	if err := c.client.Expire(ctx, fullKey, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set TTL for key %s: %w", key, err)
	}

	return nil
}

// GetTTL returns the remaining TTL of a key.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - key: The cache key
//
// Returns:
//   - time.Duration: Remaining TTL
//   - error: Any error that occurred
func (c *CacheService) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	fullKey := c.fullKey(key)

	ttl, err := c.client.TTL(ctx, fullKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL for key %s: %w", key, err)
	}

	return ttl, nil
}

// Close closes the Redis connection.
//
// Returns:
//   - error: Any error that occurred during close
func (c *CacheService) Close() error {
	return c.client.Close()
}

// Ping checks the Redis connection.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//
// Returns:
//   - error: Any error that occurred during ping
func (c *CacheService) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Client returns the underlying Redis client for advanced operations.
//
// Returns:
//   - *redis.Client: The Redis client
func (c *CacheService) Client() *redis.Client {
	return c.client
}

