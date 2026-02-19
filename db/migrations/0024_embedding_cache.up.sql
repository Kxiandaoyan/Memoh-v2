-- 0024_embedding_cache
-- Cache embedding vectors by text hash to avoid redundant API calls.
-- Shared across all bots; keyed by provider + model + text SHA-256 hash.

CREATE TABLE IF NOT EXISTS embedding_cache (
  id         UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
  provider   TEXT    NOT NULL,
  model      TEXT    NOT NULL,
  hash       TEXT    NOT NULL,
  embedding  JSONB   NOT NULL,
  dims       INT     NOT NULL,
  updated_at BIGINT  NOT NULL,
  CONSTRAINT embedding_cache_unique UNIQUE (provider, model, hash)
);

CREATE INDEX IF NOT EXISTS idx_embedding_cache_lookup  ON embedding_cache (provider, model, hash);
CREATE INDEX IF NOT EXISTS idx_embedding_cache_updated ON embedding_cache (updated_at ASC);
