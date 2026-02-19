package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	embeddingCacheMaxEntries = 50_000
	embeddingCachePruneRatio = 0.10 // prune 10% when limit is exceeded
)

// EmbeddingCache stores text→vector mappings in PostgreSQL to avoid
// redundant embedding API calls for identical text across bots.
type EmbeddingCache struct {
	pool     *pgxpool.Pool
	provider string
	model    string
	logger   *slog.Logger
}

// NewEmbeddingCache creates a cache keyed by provider and model.
func NewEmbeddingCache(pool *pgxpool.Pool, provider, model string, log *slog.Logger) *EmbeddingCache {
	return &EmbeddingCache{
		pool:     pool,
		provider: provider,
		model:    model,
		logger:   log.With(slog.String("component", "embedding_cache")),
	}
}

// HashText returns the SHA-256 hex digest of the input text, used as a cache key.
func HashText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// Get looks up a cached vector. Returns (vec, true) if found, (nil, false) otherwise.
func (c *EmbeddingCache) Get(ctx context.Context, text string) ([]float32, bool) {
	hash := HashText(text)
	row := c.pool.QueryRow(ctx,
		`SELECT embedding FROM embedding_cache WHERE provider=$1 AND model=$2 AND hash=$3`,
		c.provider, c.model, hash,
	)
	var raw []byte
	if err := row.Scan(&raw); err != nil {
		return nil, false
	}
	var vec []float32
	if err := json.Unmarshal(raw, &vec); err != nil {
		return nil, false
	}
	// Touch updated_at so LRU pruning keeps recently used entries longer.
	_, _ = c.pool.Exec(ctx,
		`UPDATE embedding_cache SET updated_at=$1 WHERE provider=$2 AND model=$3 AND hash=$4`,
		time.Now().UnixMilli(), c.provider, c.model, hash,
	)
	return vec, true
}

// Set writes a vector to the cache, upserting on conflict.
func (c *EmbeddingCache) Set(ctx context.Context, text string, vec []float32) error {
	hash := HashText(text)
	raw, err := json.Marshal(vec)
	if err != nil {
		return fmt.Errorf("marshal embedding: %w", err)
	}
	_, err = c.pool.Exec(ctx,
		`INSERT INTO embedding_cache (provider, model, hash, embedding, dims, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (provider, model, hash) DO UPDATE
		   SET embedding=$4, dims=$5, updated_at=$6`,
		c.provider, c.model, hash, raw, len(vec), time.Now().UnixMilli(),
	)
	if err != nil {
		return fmt.Errorf("upsert embedding cache: %w", err)
	}
	go c.pruneIfNeeded(context.Background())
	return nil
}

// pruneIfNeeded removes the oldest entries when the cache exceeds the max size.
func (c *EmbeddingCache) pruneIfNeeded(ctx context.Context) {
	var count int
	err := c.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM embedding_cache WHERE provider=$1 AND model=$2`,
		c.provider, c.model,
	).Scan(&count)
	if err != nil || count <= embeddingCacheMaxEntries {
		return
	}
	excess := int(float64(count) * embeddingCachePruneRatio)
	if excess < 1 {
		excess = 1
	}
	_, err = c.pool.Exec(ctx,
		`DELETE FROM embedding_cache
		 WHERE id IN (
		   SELECT id FROM embedding_cache
		   WHERE provider=$1 AND model=$2
		   ORDER BY updated_at ASC
		   LIMIT $3
		 )`,
		c.provider, c.model, excess,
	)
	if err != nil {
		c.logger.Warn("embedding cache prune failed", slog.Any("error", err))
	} else {
		c.logger.Info("embedding cache pruned", slog.Int("removed", excess))
	}
}
