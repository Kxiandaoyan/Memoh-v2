-- Drop indexes
DROP INDEX IF EXISTS idx_builtin_tool_configs_enabled;
DROP INDEX IF EXISTS idx_builtin_tool_configs_bot_id;

-- Drop table
DROP TABLE IF EXISTS builtin_tool_configs;
