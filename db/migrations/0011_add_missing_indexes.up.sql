-- 0011_add_missing_indexes
-- Add missing indexes for conversation_type and token_usage model columns.

CREATE INDEX IF NOT EXISTS idx_bot_channel_routes_conversation_type ON bot_channel_routes(conversation_type);
CREATE INDEX IF NOT EXISTS idx_token_usage_model ON token_usage(model);
