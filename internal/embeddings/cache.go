package embeddings

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

type cacheEntry struct {
	vector    []float32
	expiresAt time.Time
}

type CachedEmbedder struct {
	inner      Embedder
	modelID    string
	mu         sync.RWMutex
	cache      map[string]cacheEntry
	ttl        time.Duration
	maxEntries int
}

func NewCachedEmbedder(inner Embedder, modelID string, ttl time.Duration, maxEntries int) *CachedEmbedder {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	if maxEntries <= 0 {
		maxEntries = 10000
	}
	return &CachedEmbedder{
		inner:      inner,
		modelID:    modelID,
		cache:      make(map[string]cacheEntry, 256),
		ttl:        ttl,
		maxEntries: maxEntries,
	}
}

func (c *CachedEmbedder) Embed(ctx context.Context, input string) ([]float32, error) {
	key := c.cacheKey(input)

	c.mu.RLock()
	if entry, ok := c.cache[key]; ok && time.Now().Before(entry.expiresAt) {
		c.mu.RUnlock()
		dst := make([]float32, len(entry.vector))
		copy(dst, entry.vector)
		return dst, nil
	}
	c.mu.RUnlock()

	vector, err := c.inner.Embed(ctx, input)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	if len(c.cache) >= c.maxEntries {
		c.evictOldest()
	}
	c.cache[key] = cacheEntry{
		vector:    vector,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()

	return vector, nil
}

func (c *CachedEmbedder) Dimensions() int {
	return c.inner.Dimensions()
}

func (c *CachedEmbedder) cacheKey(input string) string {
	h := sha256.Sum256([]byte(c.modelID + ":" + input))
	return hex.EncodeToString(h[:])
}

func (c *CachedEmbedder) evictOldest() {
	now := time.Now()
	for k, v := range c.cache {
		if now.After(v.expiresAt) {
			delete(c.cache, k)
		}
	}
	if len(c.cache) >= c.maxEntries {
		count := 0
		target := len(c.cache) / 4
		for k := range c.cache {
			delete(c.cache, k)
			count++
			if count >= target {
				break
			}
		}
	}
}
