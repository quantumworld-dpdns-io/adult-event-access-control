// TODO: Register this handler in cmd/server/main.go
package cache

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// Cache defines a generic key-value cache interface.
type Cache interface {
	Get(key string) (string, error)
	Set(key string, value string, ttl time.Duration) error
	Delete(key string) error
	Exists(key string) (bool, error)
}

// inMemoryCache implements Cache with a sync.Map.
type inMemoryCache struct {
	data sync.Map
}

func (c *inMemoryCache) Get(key string) (string, error) {
	v, ok := c.data.Load(key)
	if !ok {
		return "", errors.New("cache: key not found")
	}
	return v.(string), nil
}

func (c *inMemoryCache) Set(key string, value string, ttl time.Duration) error {
	c.data.Store(key, value)
	if ttl > 0 {
		time.AfterFunc(ttl, func() {
			c.data.Delete(key)
		})
	}
	return nil
}

func (c *inMemoryCache) Delete(key string) error {
	c.data.Delete(key)
	return nil
}

func (c *inMemoryCache) Exists(key string) (bool, error) {
	_, ok := c.data.Load(key)
	return ok, nil
}

// NewInMemoryCache returns a sync.Map-backed Cache.
func NewInMemoryCache() Cache {
	return &inMemoryCache{}
}

// NewRedisCache is a stub that returns an in-memory cache.
// TODO: Replace with actual redis.Client using github.com/redis/go-redis/v9.
func NewRedisCache(host, port, password string) (Cache, error) {
	if host == "" || port == "" {
		return nil, fmt.Errorf("cache: redis host and port are required")
	}
	// Placeholder — returns in-memory fallback.
	return NewInMemoryCache(), nil
}

// typed caches wrapping the generic Cache interface.

// SeatMapCache caches seat map data.
type SeatMapCache struct {
	Cache
}

func NewSeatMapCache(store Cache) *SeatMapCache {
	return &SeatMapCache{Cache: store}
}

// WaitlistCache caches waitlist positions.
type WaitlistCache struct {
	Cache
}

func NewWaitlistCache(store Cache) *WaitlistCache {
	return &WaitlistCache{Cache: store}
}

// RateLimitCache caches rate-limit counters.
type RateLimitCache struct {
	Cache
}

func NewRateLimitCache(store Cache) *RateLimitCache {
	return &RateLimitCache{Cache: store}
}
