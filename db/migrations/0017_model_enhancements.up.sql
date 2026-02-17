-- 0017_model_enhancements
-- Add reasoning and max_tokens columns to models table, expand client_type CHECK constraint

ALTER TABLE models ADD COLUMN IF NOT EXISTS reasoning BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE models ADD COLUMN IF NOT EXISTS max_tokens INTEGER NOT NULL DEFAULT 0;

ALTER TABLE llm_providers DROP CONSTRAINT IF EXISTS llm_providers_client_type_check;
ALTER TABLE llm_providers ADD CONSTRAINT llm_providers_client_type_check CHECK (client_type IN ('openai', 'openai-compat', 'anthropic', 'google', 'azure', 'bedrock', 'mistral', 'xai', 'ollama', 'dashscope', 'deepseek', 'zai-global', 'zai-cn', 'zai-coding-global', 'zai-coding-cn', 'minimax-global', 'minimax-cn', 'moonshot-global', 'moonshot-cn', 'volcengine', 'volcengine-coding', 'qianfan', 'groq', 'openrouter', 'together', 'fireworks', 'perplexity'));
