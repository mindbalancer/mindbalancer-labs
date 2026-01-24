// Package cache provides response caching for LLM requests.
package cache

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
)

// Config holds cache configuration.
type Config struct {
	Enabled     bool
	MaxSize     int           // Maximum number of cached items
	TTL         time.Duration // Time to live for cache entries
	MaxItemSize int64         // Maximum size of a single cached item in bytes
}

// DefaultConfig returns default cache configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:     true,
		MaxSize:     1000,
		TTL:         5 * time.Minute,
		MaxItemSize: 1024 * 1024, // 1MB
	}
}

// Entry represents a cached response.
type Entry struct {
	Key       string
	Value     []byte
	Model     string
	CreatedAt time.Time
	ExpiresAt time.Time
	HitCount  int64
	Size      int64
}

// IsExpired returns true if the entry has expired.
func (e *Entry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// Cache implements an LRU cache with TTL for LLM responses.
type Cache struct {
	mu       sync.RWMutex
	cfg      Config
	items    map[string]*list.Element
	lru      *list.List
	stats    Stats
}

// Stats holds cache statistics.
type Stats struct {
	Hits       int64
	Misses     int64
	Evictions  int64
	Size       int64
	ItemCount  int
}

// NewCache creates a new cache.
func NewCache(cfg Config) *Cache {
	c := &Cache{
		cfg:   cfg,
		items: make(map[string]*list.Element),
		lru:   list.New(),
	}
	
	// Start cleanup goroutine
	if cfg.Enabled {
		go c.cleanupLoop()
	}
	
	return c
}

// GenerateKey creates a cache key from a request.
func GenerateKey(model string, messages any, temperature *float64, maxTokens *int) string {
	h := sha256.New()
	
	// Include model
	h.Write([]byte(model))
	
	// Include messages
	if messages != nil {
		msgBytes, _ := json.Marshal(messages)
		h.Write(msgBytes)
	}
	
	// Include temperature (affects output)
	if temperature != nil {
		h.Write([]byte{byte(*temperature * 100)})
	}
	
	// Include max_tokens
	if maxTokens != nil {
		h.Write([]byte{byte(*maxTokens >> 8), byte(*maxTokens)})
	}
	
	return hex.EncodeToString(h.Sum(nil))
}

// Get retrieves a cached response.
func (c *Cache) Get(key string) ([]byte, bool) {
	if !c.cfg.Enabled {
		return nil, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		c.stats.Misses++
		return nil, false
	}

	entry := elem.Value.(*Entry)
	if entry.IsExpired() {
		c.removeElement(elem)
		c.stats.Misses++
		return nil, false
	}

	// Move to front (LRU)
	c.lru.MoveToFront(elem)
	entry.HitCount++
	c.stats.Hits++

	return entry.Value, true
}

// Set stores a response in the cache.
func (c *Cache) Set(key string, value []byte, model string) {
	if !c.cfg.Enabled {
		return
	}

	size := int64(len(value))
	if size > c.cfg.MaxItemSize {
		return // Item too large
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if key exists
	if elem, ok := c.items[key]; ok {
		c.lru.MoveToFront(elem)
		entry := elem.Value.(*Entry)
		c.stats.Size -= entry.Size
		entry.Value = value
		entry.Size = size
		entry.ExpiresAt = time.Now().Add(c.cfg.TTL)
		c.stats.Size += size
		return
	}

	// Evict if necessary
	for c.lru.Len() >= c.cfg.MaxSize {
		c.evictOldest()
	}

	// Add new entry
	entry := &Entry{
		Key:       key,
		Value:     value,
		Model:     model,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(c.cfg.TTL),
		HitCount:  0,
		Size:      size,
	}

	elem := c.lru.PushFront(entry)
	c.items[key] = elem
	c.stats.Size += size
	c.stats.ItemCount++
}

// Delete removes an entry from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.removeElement(elem)
	}
}

// Clear removes all entries from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.lru.Init()
	c.stats.Size = 0
	c.stats.ItemCount = 0
}

// Stats returns cache statistics.
func (c *Cache) Stats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	stats := c.stats
	stats.ItemCount = c.lru.Len()
	return stats
}

// HitRate returns the cache hit rate.
func (c *Cache) HitRate() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.stats.Hits + c.stats.Misses
	if total == 0 {
		return 0
	}
	return float64(c.stats.Hits) / float64(total)
}

func (c *Cache) removeElement(elem *list.Element) {
	entry := elem.Value.(*Entry)
	delete(c.items, entry.Key)
	c.lru.Remove(elem)
	c.stats.Size -= entry.Size
	c.stats.ItemCount--
}

func (c *Cache) evictOldest() {
	elem := c.lru.Back()
	if elem != nil {
		c.removeElement(elem)
		c.stats.Evictions++
	}
}

func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

func (c *Cache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	var toRemove []*list.Element
	
	for elem := c.lru.Back(); elem != nil; elem = elem.Prev() {
		entry := elem.Value.(*Entry)
		if entry.IsExpired() {
			toRemove = append(toRemove, elem)
		}
	}

	for _, elem := range toRemove {
		c.removeElement(elem)
		c.stats.Evictions++
	}
}

// SetEnabled enables or disables caching.
func (c *Cache) SetEnabled(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cfg.Enabled = enabled
}

// IsEnabled returns whether caching is enabled.
func (c *Cache) IsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cfg.Enabled
}

// Global cache instance
var (
	globalCache     *Cache
	globalCacheOnce sync.Once
)

// InitGlobalCache initializes the global cache.
func InitGlobalCache(cfg Config) {
	globalCacheOnce = sync.Once{}
	globalCacheOnce.Do(func() {
		globalCache = NewCache(cfg)
	})
}

// GlobalCache returns the global cache instance.
func GlobalCache() *Cache {
	globalCacheOnce.Do(func() {
		globalCache = NewCache(DefaultConfig())
	})
	return globalCache
}
