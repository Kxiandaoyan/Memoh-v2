-- 0013_add_serpapi_provider (down)
-- Revert to brave-only CHECK constraint. Rows with provider='serpapi' must be deleted first.

DELETE FROM search_providers WHERE provider = 'serpapi';
ALTER TABLE search_providers DROP CONSTRAINT IF EXISTS search_providers_provider_check;
ALTER TABLE search_providers ADD CONSTRAINT search_providers_provider_check CHECK (provider IN ('brave'));
