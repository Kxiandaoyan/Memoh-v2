-- 0034_embedding_cache_ttl
-- Add TTL mechanism to embedding cache to prevent using stale embeddings after model updates.

ALTER TABLE embedding_cache ADD COLUMN IF NOT EXISTS expires_at BIGINT;

UPDATE embedding_cache
SET expires_at = EXTRACT(EPOCH FROM (NOW() + INTERVAL '7 days')) * 1000
WHERE expires_at IS NULL;

ALTER TABLE embedding_cache ALTER COLUMN expires_at SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_embedding_cache_expires ON embedding_cache (expires_at ASC);
