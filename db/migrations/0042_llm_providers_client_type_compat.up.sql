-- 0042_llm_providers_client_type_compat
-- Add client_type for legacy llm_providers rows and enforce valid values.

ALTER TABLE llm_providers
  ADD COLUMN IF NOT EXISTS client_type TEXT;

UPDATE llm_providers
SET client_type = CASE
  WHEN client_type IS NULL OR btrim(client_type) = '' THEN 'openai-compat'
  WHEN client_type IN (
    'openai', 'openai-compat', 'anthropic', 'google', 'azure', 'bedrock',
    'mistral', 'xai', 'ollama', 'dashscope', 'deepseek', 'zai-global',
    'zai-cn', 'zai-coding-global', 'zai-coding-cn', 'minimax-global',
    'minimax-cn', 'moonshot-global', 'moonshot-cn', 'volcengine',
    'volcengine-coding', 'qianfan', 'groq', 'openrouter', 'together',
    'fireworks', 'perplexity'
  ) THEN client_type
  ELSE 'openai-compat'
END;

ALTER TABLE llm_providers
  ALTER COLUMN client_type SET DEFAULT 'openai-compat';

ALTER TABLE llm_providers
  ALTER COLUMN client_type SET NOT NULL;

ALTER TABLE llm_providers
  DROP CONSTRAINT IF EXISTS llm_providers_client_type_check;

ALTER TABLE llm_providers
  ADD CONSTRAINT llm_providers_client_type_check CHECK (
    client_type IN (
      'openai', 'openai-compat', 'anthropic', 'google', 'azure', 'bedrock',
      'mistral', 'xai', 'ollama', 'dashscope', 'deepseek', 'zai-global',
      'zai-cn', 'zai-coding-global', 'zai-coding-cn', 'minimax-global',
      'minimax-cn', 'moonshot-global', 'moonshot-cn', 'volcengine',
      'volcengine-coding', 'qianfan', 'groq', 'openrouter', 'together',
      'fireworks', 'perplexity'
    )
  );
