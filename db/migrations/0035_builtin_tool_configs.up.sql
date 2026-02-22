-- builtin_tool_configs table: stores per-bot configuration for builtin tools
CREATE TABLE IF NOT EXISTS builtin_tool_configs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
  tool_name TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT true,
  priority INTEGER NOT NULL DEFAULT 100,
  category TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT builtin_tool_configs_unique UNIQUE (bot_id, tool_name)
);

CREATE INDEX IF NOT EXISTS idx_builtin_tool_configs_bot_id ON builtin_tool_configs(bot_id);
CREATE INDEX IF NOT EXISTS idx_builtin_tool_configs_enabled ON builtin_tool_configs(bot_id, enabled) WHERE enabled = true;
