-- 0017_model_enhancements (rollback)
-- Remove reasoning and max_tokens columns, restore original client_type CHECK constraint

ALTER TABLE models DROP COLUMN IF EXISTS reasoning;
ALTER TABLE models DROP COLUMN IF EXISTS max_tokens;

ALTER TABLE llm_providers DROP CONSTRAINT IF EXISTS llm_providers_client_type_check;
ALTER TABLE llm_providers ADD CONSTRAINT llm_providers_client_type_check CHECK (client_type IN ('openai', 'openai-compat', 'anthropic', 'google', 'azure', 'bedrock', 'mistral', 'xai', 'ollama', 'dashscope'));
