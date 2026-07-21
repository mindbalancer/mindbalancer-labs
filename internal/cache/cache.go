// Package cache provides advanced response caching for LLM requests.
package cache

import (
	"bytes"
	"compress/gzip"
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"hash/fnv"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// Config holds cache configuration.
type Config struct {
	Enabled            bool
	MaxSize            int           // Maximum number of cached items (per shard)
	MaxMemoryMB        int           // Maximum total memory in MB (0 = unlimited)
	TTL                time.Duration // Default time to live for cache entries
	MaxItemSize        int64         // Maximum size of a single cached item in bytes
	NumShards          int           // Number of cache shards (default: 16)
	CompressionEnabled bool          // Enable gzip compression for large items
	CompressionMinSize int64         // Minimum size to trigger compression (bytes)
	CleanupInterval    time.Duration // Interval for cleanup goroutine

	// Model-specific TTLs (model prefix -> TTL)
	ModelTTLs map[string]time.Duration

	// Endpoint-specific TTLs
	EmbeddingsTTL time.Duration // TTL for embeddings (usually longer)
}

// DefaultConfig returns default cache configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:            true,
		MaxSize:            1000,
		MaxMemoryMB:        512, // 512MB default
		TTL:                5 * time.Minute,
		MaxItemSize:        2 * 1024 * 1024, // 2MB
		NumShards:          16,
		CompressionEnabled: true,
		CompressionMinSize: 1024, // Compress items > 1KB
		CleanupInterval:    time.Minute,
		ModelTTLs: map[string]time.Duration{
			"text-embedding": 24 * time.Hour, // Embeddings are deterministic
			"embedding":      24 * time.Hour,
		},
		EmbeddingsTTL: 24 * time.Hour,
	}
}

// Entry represents a cached response.
type Entry struct {
	Key        string
	Value      []byte
	Model      string
	Endpoint   string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	LastAccess time.Time
	HitCount   int64
	Size       int64 // Original size
	StoredSize int64 // Size after compression
	Compressed bool
}

// IsExpired returns true if the entry has expired.
func (e *Entry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// shard represents a single cache shard with its own lock.
type shard struct {
	mu    sync.RWMutex
	items map[string]*list.Element
	lru   *list.List
	size  int64 // Current memory usage in bytes
}

// Stats holds cache statistics.
type Stats struct {
	Hits             int64   `json:"hits"`
	Misses           int64   `json:"misses"`
	Evictions        int64   `json:"evictions"`
	DeduplicatedReqs int64   `json:"deduplicated_requests"`
	CompressionSaved int64   `json:"compression_saved_bytes"`
	MemoryUsed       int64   `json:"memory_used_bytes"`
	ItemCount        int     `json:"item_count"`
	HitRate          float64 `json:"hit_rate"`
	AvgItemSize      float64 `json:"avg_item_size_bytes"`
}

// inflight represents an in-flight request for deduplication.
type inflight struct {
	done   chan struct{}
	result []byte
	err    error
}

// Cache implements a sharded LRU cache with TTL, compression, and request deduplication.
type Cache struct {
	cfg      Config
	shards   []*shard
	stats    cacheStats
	inflight sync.Map // map[string]*inflight - for request deduplication
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// cacheStats holds atomic counters for statistics.
type cacheStats struct {
	hits             atomic.Int64
	misses           atomic.Int64
	evictions        atomic.Int64
	deduplicatedReqs atomic.Int64
	compressionSaved atomic.Int64
}

// NewCache creates a new sharded cache.
func NewCache(cfg Config) *Cache {
	if cfg.NumShards <= 0 {
		cfg.NumShards = 16
	}

	c := &Cache{
		cfg:    cfg,
		shards: make([]*shard, cfg.NumShards),
		stopCh: make(chan struct{}),
	}

	for i := 0; i < cfg.NumShards; i++ {
		c.shards[i] = &shard{
			items: make(map[string]*list.Element),
			lru:   list.New(),
		}
	}

	// Start cleanup goroutine
	if cfg.Enabled {
		c.wg.Add(1)
		go c.cleanupLoop()
	}

	return c
}

// getShard returns the shard for a given key.
func (c *Cache) getShard(key string) *shard {
	h := fnv.New32a()
	h.Write([]byte(key))
	return c.shards[h.Sum32()%uint32(len(c.shards))]
}

// GenerateKey creates a cache key from a request.
func GenerateKey(endpoint, model string, body any) string {
	h := sha256.New()

	// Include endpoint
	h.Write([]byte(endpoint))
	h.Write([]byte{0}) // separator

	// Include model
	h.Write([]byte(model))
	h.Write([]byte{0})

	// Include request body
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		h.Write(bodyBytes)
	}

	return hex.EncodeToString(h.Sum(nil))
}

// GenerateChatKey creates a cache key specifically for chat completions.
func GenerateChatKey(model string, messages any, temperature *float64, maxTokens *int, systemFingerprint string) string {
	h := sha256.New()

	// Include model
	h.Write([]byte(model))
	h.Write([]byte{0})

	// Include messages (normalized)
	if messages != nil {
		msgBytes, _ := json.Marshal(messages)
		h.Write(msgBytes)
	}
	h.Write([]byte{0})

	// Include temperature (affects output)
	if temperature != nil {
		// Use fixed-point representation for consistency
		tempInt := int64(*temperature * 1000)
		h.Write([]byte{byte(tempInt >> 24), byte(tempInt >> 16), byte(tempInt >> 8), byte(tempInt)})
	}
	h.Write([]byte{0})

	// Include max_tokens
	if maxTokens != nil {
		h.Write([]byte{byte(*maxTokens >> 24), byte(*maxTokens >> 16), byte(*maxTokens >> 8), byte(*maxTokens)})
	}

	// Include system fingerprint if available (for reproducibility)
	if systemFingerprint != "" {
		h.Write([]byte{0})
		h.Write([]byte(systemFingerprint))
	}

	return hex.EncodeToString(h.Sum(nil))
}

// GenerateEmbeddingKey creates a cache key for embeddings.
func GenerateEmbeddingKey(model string, input any, dimensions *int) string {
	h := sha256.New()

	h.Write([]byte("embedding"))
	h.Write([]byte{0})
	h.Write([]byte(model))
	h.Write([]byte{0})

	// Normalize input
	inputBytes, _ := json.Marshal(input)
	h.Write(inputBytes)

	if dimensions != nil {
		h.Write([]byte{0})
		h.Write([]byte{byte(*dimensions >> 8), byte(*dimensions)})
	}

	return hex.EncodeToString(h.Sum(nil))
}

// Get retrieves a cached response.
func (c *Cache) Get(key string) ([]byte, bool) {
	if !c.cfg.Enabled {
		return nil, false
	}

	shard := c.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	elem, ok := shard.items[key]
	if !ok {
		c.stats.misses.Add(1)
		return nil, false
	}

	entry := elem.Value.(*Entry)
	if entry.IsExpired() {
		c.removeFromShard(shard, elem)
		c.stats.misses.Add(1)
		return nil, false
	}

	// Move to front (LRU)
	shard.lru.MoveToFront(elem)
	entry.HitCount++
	entry.LastAccess = time.Now()
	c.stats.hits.Add(1)

	// Decompress if needed
	value := entry.Value
	if entry.Compressed {
		decompressed, err := c.decompress(entry.Value)
		if err != nil {
			// Corruption - remove entry
			c.removeFromShard(shard, elem)
			return nil, false
		}
		value = decompressed
	}

	return value, true
}

// Set stores a response in the cache.
func (c *Cache) Set(key string, value []byte, model, endpoint string) {
	c.SetWithTTL(key, value, model, endpoint, 0)
}

// SetWithTTL stores a response with a specific TTL.
func (c *Cache) SetWithTTL(key string, value []byte, model, endpoint string, ttl time.Duration) {
	if !c.cfg.Enabled {
		return
	}

	originalSize := int64(len(value))
	if originalSize > c.cfg.MaxItemSize {
		return // Item too large
	}

	// Determine TTL
	if ttl == 0 {
		ttl = c.getTTL(model, endpoint)
	}

	// Compress if enabled and size warrants it
	storedValue := value
	compressed := false
	storedSize := originalSize

	if c.cfg.CompressionEnabled && originalSize >= c.cfg.CompressionMinSize {
		compressedValue, err := c.compress(value)
		if err == nil && int64(len(compressedValue)) < originalSize {
			storedValue = compressedValue
			compressed = true
			storedSize = int64(len(compressedValue))
			c.stats.compressionSaved.Add(originalSize - storedSize)
		}
	}

	shard := c.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	now := time.Now()

	// Check if key exists - update it
	if elem, ok := shard.items[key]; ok {
		shard.lru.MoveToFront(elem)
		entry := elem.Value.(*Entry)
		shard.size -= entry.StoredSize
		entry.Value = storedValue
		entry.Size = originalSize
		entry.StoredSize = storedSize
		entry.Compressed = compressed
		entry.ExpiresAt = now.Add(ttl)
		entry.LastAccess = now
		shard.size += storedSize
		return
	}

	// Check memory limit and evict if necessary
	c.evictIfNeeded(shard, storedSize)

	// Evict by count if necessary
	maxPerShard := c.cfg.MaxSize / c.cfg.NumShards
	if maxPerShard < 10 {
		maxPerShard = 10
	}
	for shard.lru.Len() >= maxPerShard {
		c.evictOldestFromShard(shard)
	}

	// Add new entry
	entry := &Entry{
		Key:        key,
		Value:      storedValue,
		Model:      model,
		Endpoint:   endpoint,
		CreatedAt:  now,
		ExpiresAt:  now.Add(ttl),
		LastAccess: now,
		HitCount:   0,
		Size:       originalSize,
		StoredSize: storedSize,
		Compressed: compressed,
	}

	elem := shard.lru.PushFront(entry)
	shard.items[key] = elem
	shard.size += storedSize
}

// getTTL returns the appropriate TTL for a model/endpoint.
func (c *Cache) getTTL(model, endpoint string) time.Duration {
	// Check endpoint-specific TTL
	if endpoint == "/v1/embeddings" && c.cfg.EmbeddingsTTL > 0 {
		return c.cfg.EmbeddingsTTL
	}

	// Check model-specific TTLs
	for prefix, ttl := range c.cfg.ModelTTLs {
		if len(model) >= len(prefix) && model[:len(prefix)] == prefix {
			return ttl
		}
	}

	return c.cfg.TTL
}

// evictIfNeeded evicts entries if memory limit is exceeded.
func (c *Cache) evictIfNeeded(shard *shard, newSize int64) {
	if c.cfg.MaxMemoryMB <= 0 {
		return
	}

	maxMemory := int64(c.cfg.MaxMemoryMB) * 1024 * 1024 / int64(c.cfg.NumShards)

	for shard.size+newSize > maxMemory && shard.lru.Len() > 0 {
		c.evictOldestFromShard(shard)
	}
}

// evictOldestFromShard removes the oldest entry from a shard.
func (c *Cache) evictOldestFromShard(shard *shard) {
	elem := shard.lru.Back()
	if elem != nil {
		c.removeFromShard(shard, elem)
		c.stats.evictions.Add(1)
	}
}

// removeFromShard removes an element from a shard.
func (c *Cache) removeFromShard(shard *shard, elem *list.Element) {
	entry := elem.Value.(*Entry)
	delete(shard.items, entry.Key)
	shard.lru.Remove(elem)
	shard.size -= entry.StoredSize
}

// Delete removes an entry from the cache.
func (c *Cache) Delete(key string) {
	shard := c.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if elem, ok := shard.items[key]; ok {
		c.removeFromShard(shard, elem)
	}
}

// Clear removes all entries from the cache.
func (c *Cache) Clear() {
	for _, shard := range c.shards {
		shard.mu.Lock()
		shard.items = make(map[string]*list.Element)
		shard.lru.Init()
		shard.size = 0
		shard.mu.Unlock()
	}
}

// Stats returns cache statistics.
func (c *Cache) Stats() Stats {
	var totalSize int64
	var totalCount int

	for _, shard := range c.shards {
		shard.mu.RLock()
		totalSize += shard.size
		totalCount += shard.lru.Len()
		shard.mu.RUnlock()
	}

	hits := c.stats.hits.Load()
	misses := c.stats.misses.Load()
	total := hits + misses

	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	var avgItemSize float64
	if totalCount > 0 {
		avgItemSize = float64(totalSize) / float64(totalCount)
	}

	return Stats{
		Hits:             hits,
		Misses:           misses,
		Evictions:        c.stats.evictions.Load(),
		DeduplicatedReqs: c.stats.deduplicatedReqs.Load(),
		CompressionSaved: c.stats.compressionSaved.Load(),
		MemoryUsed:       totalSize,
		ItemCount:        totalCount,
		HitRate:          hitRate,
		AvgItemSize:      avgItemSize,
	}
}

// HitRate returns the cache hit rate.
func (c *Cache) HitRate() float64 {
	hits := c.stats.hits.Load()
	misses := c.stats.misses.Load()
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total)
}

// ===== Request Deduplication =====

// DeduplicatedCall executes fn only once for concurrent identical requests.
// Other callers with the same key will wait and receive the same result.
func (c *Cache) DeduplicatedCall(key string, fn func() ([]byte, error)) ([]byte, error, bool) {
	if !c.cfg.Enabled {
		result, err := fn()
		return result, err, false
	}

	// Check if there's already an in-flight request
	newFlight := &inflight{done: make(chan struct{})}

	if existing, loaded := c.inflight.LoadOrStore(key, newFlight); loaded {
		// Another request is in flight - wait for it
		flight := existing.(*inflight)
		<-flight.done
		c.stats.deduplicatedReqs.Add(1)
		return flight.result, flight.err, true // true = was deduplicated
	}

	// We're the first - execute the function
	result, err := fn()

	// Store result and signal waiters
	newFlight.result = result
	newFlight.err = err
	close(newFlight.done)

	// Clean up after a short delay to catch late arrivals
	go func() {
		time.Sleep(100 * time.Millisecond)
		c.inflight.Delete(key)
	}()

	return result, err, false
}

// ===== Compression =====

func (c *Cache) compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	if err != nil {
		return nil, err
	}

	if _, err := gz.Write(data); err != nil {
		gz.Close()
		return nil, err
	}

	if err := gz.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (c *Cache) decompress(data []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	return io.ReadAll(gz)
}

// ===== Cleanup =====

func (c *Cache) cleanupLoop() {
	defer c.wg.Done()

	interval := c.cfg.CleanupInterval
	if interval <= 0 {
		interval = time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCh:
			return
		}
	}
}

func (c *Cache) cleanup() {
	for _, shard := range c.shards {
		c.cleanupShard(shard)
	}
}

func (c *Cache) cleanupShard(shard *shard) {
	shard.mu.Lock()
	defer shard.mu.Unlock()

	var toRemove []*list.Element

	for elem := shard.lru.Back(); elem != nil; elem = elem.Prev() {
		entry := elem.Value.(*Entry)
		if entry.IsExpired() {
			toRemove = append(toRemove, elem)
		}
	}

	for _, elem := range toRemove {
		c.removeFromShard(shard, elem)
		c.stats.evictions.Add(1)
	}
}

// ===== Lifecycle =====

// Stop gracefully stops the cache cleanup goroutine.
func (c *Cache) Stop() {
	close(c.stopCh)
	c.wg.Wait()
}

// SetEnabled enables or disables caching.
func (c *Cache) SetEnabled(enabled bool) {
	c.cfg.Enabled = enabled
}

// IsEnabled returns whether caching is enabled.
func (c *Cache) IsEnabled() bool {
	return c.cfg.Enabled
}

// GetConfig returns the current cache configuration.
func (c *Cache) GetConfig() Config {
	return c.cfg
}

// UpdateConfig updates cache configuration at runtime.
func (c *Cache) UpdateConfig(cfg Config) {
	c.cfg.Enabled = cfg.Enabled
	c.cfg.TTL = cfg.TTL
	c.cfg.MaxItemSize = cfg.MaxItemSize
	c.cfg.MaxMemoryMB = cfg.MaxMemoryMB
	c.cfg.CompressionEnabled = cfg.CompressionEnabled
	// Note: NumShards cannot be changed at runtime
}

// ===== Global cache instance =====

var (
	globalCache     *Cache
	globalCacheOnce sync.Once
	globalCacheMu   sync.Mutex
)

// InitGlobalCache initializes the global cache.
func InitGlobalCache(cfg Config) {
	globalCacheMu.Lock()
	defer globalCacheMu.Unlock()

	if globalCache != nil {
		globalCache.Stop()
	}

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
