package internal

import (
	"fmt"
	"sync"
	"time"
)

type cacheItem struct {
	value     interface{}
	timestamp time.Time
}

type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]cacheItem
	ttl   time.Duration
}

var (
	// 全局缓存实例
	readabilityCache = NewMemoryCache(time.Hour)
)

func NewMemoryCache(ttl time.Duration) *MemoryCache {
	cache := &MemoryCache{
		items: make(map[string]cacheItem),
		ttl:   ttl,
	}

	go cache.cleanupLoop()

	return cache
}

func genCacheKey(scene string, key interface{}) string {
	return fmt.Sprintf("%s-%v", scene, key)
}

func (c *MemoryCache) Set(scene string, key interface{}, value interface{}) {
	cacheKey := genCacheKey(scene, key)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[cacheKey] = cacheItem{
		value:     value,
		timestamp: time.Now(),
	}
}

func (c *MemoryCache) Get(scene string, key interface{}) (interface{}, bool) {
	cacheKey := genCacheKey(scene, key)
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[cacheKey]
	if !exists {
		return nil, false
	}

	if time.Since(item.timestamp) > c.ttl {
		return nil, false
	}

	return item.value, true
}

func (c *MemoryCache) Delete(scene string, key interface{}) {
	cacheKey := genCacheKey(scene, key)
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, cacheKey)
}

func (c *MemoryCache) cleanupLoop() {
	ticker := time.NewTicker(c.ttl)
	for range ticker.C {
		c.cleanup()
	}
}

func (c *MemoryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if now.Sub(item.timestamp) > c.ttl {
			delete(c.items, key)
		}
	}
}
