-- 0034_embedding_cache_ttl (rollback)
DROP INDEX IF EXISTS idx_embedding_cache_expires;
ALTER TABLE embedding_cache DROP COLUMN IF EXISTS expires_at;
