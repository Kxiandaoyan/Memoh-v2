package skills

import (
	"sync"
	"time"
)

// CacheEntry represents a cached skill with metadata.
type CacheEntry struct {
	Content    string
	Metadata   map[string]any
	LoadedAt   time.Time
	AccessedAt time.Time
	HitCount   int
}

// SkillCache provides thread-safe caching for loaded skills.
// This reduces file I/O operations and improves performance when
// loading skills repeatedly.
type SkillCache struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
	maxSize int
	ttl     time.Duration
}

// NewSkillCache creates a new skill cache with the specified maximum size and TTL.
// maxSize: maximum number of skills to cache (0 = unlimited)
// ttl: time-to-live for cache entries (0 = no expiration)
func NewSkillCache(maxSize int, ttl time.Duration) *SkillCache {
	return &SkillCache{
		entries: make(map[string]*CacheEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// Get retrieves a skill from the cache.
// Returns the cache entry and true if found and not expired, nil and false otherwise.
func (c *SkillCache) Get(key string) (*CacheEntry, bool) {
	c.mu.RLock()
	entry, exists := c.entries[key]
	c.mu.RUnlock()

	if !exists {
		return nil, false
	}

	// Check if entry is expired
	if c.ttl > 0 && time.Since(entry.LoadedAt) > c.ttl {
		c.Delete(key)
		return nil, false
	}

	// Update access time and hit count
	c.mu.Lock()
	entry.AccessedAt = time.Now()
	entry.HitCount++
	c.mu.Unlock()

	return entry, true
}

// Set adds or updates a skill in the cache.
func (c *SkillCache) Set(key string, content string, metadata map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict oldest entries if cache is full
	if c.maxSize > 0 && len(c.entries) >= c.maxSize {
		c.evictLRU()
	}

	now := time.Now()
	c.entries[key] = &CacheEntry{
		Content:    content,
		Metadata:   metadata,
		LoadedAt:   now,
		AccessedAt: now,
		HitCount:   0,
	}
}

// Delete removes a skill from the cache.
func (c *SkillCache) Delete(key string) {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}

// Clear removes all entries from the cache.
func (c *SkillCache) Clear() {
	c.mu.Lock()
	c.entries = make(map[string]*CacheEntry)
	c.mu.Unlock()
}

// Size returns the current number of entries in the cache.
func (c *SkillCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// evictLRU removes the least recently used entry from the cache.
// Must be called with lock held.
func (c *SkillCache) evictLRU() {
	if len(c.entries) == 0 {
		return
	}

	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.entries {
		if oldestKey == "" || entry.AccessedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.AccessedAt
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

// CleanupExpired removes all expired entries from the cache.
func (c *SkillCache) CleanupExpired() int {
	if c.ttl == 0 {
		return 0
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	removed := 0
	now := time.Now()

	for key, entry := range c.entries {
		if now.Sub(entry.LoadedAt) > c.ttl {
			delete(c.entries, key)
			removed++
		}
	}

	return removed
}

// Stats returns cache statistics.
func (c *SkillCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalHits := 0
	for _, entry := range c.entries {
		totalHits += entry.HitCount
	}

	return CacheStats{
		Size:      len(c.entries),
		TotalHits: totalHits,
		MaxSize:   c.maxSize,
		TTL:       c.ttl,
	}
}

// CacheStats represents cache statistics.
type CacheStats struct {
	Size      int
	TotalHits int
	MaxSize   int
	TTL       time.Duration
}

// LazyLoader provides lazy loading functionality for skills.
// Skills are only loaded from disk when first accessed.
type LazyLoader struct {
	cache      *SkillCache
	loadFunc   func(skillName string) (string, map[string]any, error)
	mu         sync.RWMutex
	loading    map[string]*sync.Mutex // Track currently loading skills
	loadingMu  sync.Mutex
}

// NewLazyLoader creates a new lazy loader with the specified cache and load function.
func NewLazyLoader(cache *SkillCache, loadFunc func(skillName string) (string, map[string]any, error)) *LazyLoader {
	return &LazyLoader{
		cache:    cache,
		loadFunc: loadFunc,
		loading:  make(map[string]*sync.Mutex),
	}
}

// Load loads a skill lazily, using the cache if available.
func (l *LazyLoader) Load(skillName string) (string, map[string]any, error) {
	// Try cache first
	if entry, found := l.cache.Get(skillName); found {
		return entry.Content, entry.Metadata, nil
	}

	// Prevent duplicate loads of the same skill
	l.loadingMu.Lock()
	skillMu, isLoading := l.loading[skillName]
	if !isLoading {
		skillMu = &sync.Mutex{}
		l.loading[skillName] = skillMu
	}
	l.loadingMu.Unlock()

	// If already loading, wait for it to complete
	skillMu.Lock()
	defer skillMu.Unlock()

	// Check cache again in case another goroutine loaded it
	if entry, found := l.cache.Get(skillName); found {
		return entry.Content, entry.Metadata, nil
	}

	// Load from disk
	content, metadata, err := l.loadFunc(skillName)
	if err != nil {
		// Clean up loading tracker
		l.loadingMu.Lock()
		delete(l.loading, skillName)
		l.loadingMu.Unlock()
		return "", nil, err
	}

	// Store in cache
	l.cache.Set(skillName, content, metadata)

	// Clean up loading tracker
	l.loadingMu.Lock()
	delete(l.loading, skillName)
	l.loadingMu.Unlock()

	return content, metadata, nil
}

// Invalidate removes a skill from the cache, forcing it to be reloaded on next access.
func (l *LazyLoader) Invalidate(skillName string) {
	l.cache.Delete(skillName)
}

// InvalidateAll clears all cached skills.
func (l *LazyLoader) InvalidateAll() {
	l.cache.Clear()
}
