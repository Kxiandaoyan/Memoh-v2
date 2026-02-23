-- Add knowledge tools for all existing bots
INSERT INTO builtin_tool_configs (bot_id, tool_name, enabled, priority, category)
SELECT id, 'knowledge_read', true, 100, 'knowledge' FROM bots
ON CONFLICT (bot_id, tool_name) DO NOTHING;

INSERT INTO builtin_tool_configs (bot_id, tool_name, enabled, priority, category)
SELECT id, 'knowledge_write', true, 100, 'knowledge' FROM bots
ON CONFLICT (bot_id, tool_name) DO NOTHING;
