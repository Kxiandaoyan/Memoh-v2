-- 0013_add_serpapi_provider
-- Extend search_providers.provider CHECK constraint to allow 'serpapi'.

ALTER TABLE search_providers DROP CONSTRAINT IF EXISTS search_providers_provider_check;
ALTER TABLE search_providers ADD CONSTRAINT search_providers_provider_check CHECK (provider IN ('brave', 'serpapi'));
