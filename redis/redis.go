package gofacades

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrKeyNotFound = errors.New("key not found in cache")
	ErrNilCallback = errors.New("callback function cannot be nil")
)

// Client represents a Redis client
type Client struct {
	client *redis.Client
}

// Config holds the configuration for Redis connection
type Config struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// New creates a new Redis client
func New(cfg Config) (*Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test the connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	return &Client{
		client: client,
	}, nil
}

// Get retrieves an item from the cache by key
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	value, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrKeyNotFound
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

// Has checks if an item exists in the cache
func (c *Client) Has(ctx context.Context, key string) (bool, error) {
	exists, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// Remember gets an item from the cache, or stores the result of the callback
func (c *Client) Remember(ctx context.Context, key string, ttl time.Duration, callback func() (interface{}, error)) (string, error) {
	// First, try to get the existing item
	value, err := c.Get(ctx, key)
	if err == nil {
		return value, nil
	}
	if !errors.Is(err, ErrKeyNotFound) {
		return "", err
	}

	// If callback is nil, return error
	if callback == nil {
		return "", ErrNilCallback
	}

	// Execute callback
	result, err := callback()
	if err != nil {
		return "", fmt.Errorf("callback execution failed: %w", err)
	}

	// Marshal the result to JSON string
	jsonValue, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal callback result: %w", err)
	}

	// Store the result in cache
	err = c.Put(ctx, key, string(jsonValue), ttl)
	if err != nil {
		return "", err
	}

	return string(jsonValue), nil
}

// Pull retrieves and deletes an item from the cache
func (c *Client) Pull(ctx context.Context, key string) (string, error) {
	// Get the value first
	value, err := c.Get(ctx, key)
	if err != nil {
		return "", err
	}

	// Then delete it
	err = c.Forget(ctx, key)
	if err != nil {
		return "", err
	}

	return value, nil
}

// Put stores an item in the cache for a given duration
func (c *Client) Put(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

// Forever stores an item in the cache permanently
func (c *Client) Forever(ctx context.Context, key, value string) error {
	return c.client.Set(ctx, key, value, 0).Err()
}

// Forget removes an item from the cache
func (c *Client) Forget(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// Flush removes all items from the cache
func (c *Client) Flush(ctx context.Context) error {
	return c.client.FlushAll(ctx).Err()
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.client.Close()
}
