-- 0011_add_missing_indexes (down)
-- Remove indexes added in 0011.

DROP INDEX IF EXISTS idx_token_usage_model;
DROP INDEX IF EXISTS idx_bot_channel_routes_conversation_type;
